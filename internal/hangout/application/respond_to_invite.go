package application

import (
	"context"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type RespondToInviteUseCase struct {
	participants domain.ParticipantRepository
	notifier     domain.Notifier
}

func NewRespondToInviteUseCase(participants domain.ParticipantRepository, notifier domain.Notifier) *RespondToInviteUseCase {
	return &RespondToInviteUseCase{participants: participants, notifier: notifier}
}

type RespondToInviteInput struct {
	HangoutID uuid.UUID
	UserID    uuid.UUID
	Accept    bool
	// Reason is optional context for a rejection (e.g. "busy that
	// day"). Ignored when Accept is true.
	Reason *string
}

func (uc *RespondToInviteUseCase) Execute(ctx context.Context, in RespondToInviteInput) (*domain.Participant, error) {
	participant, err := uc.participants.FindParticipant(ctx, in.HangoutID, in.UserID)
	if err != nil {
		return nil, err
	}
	if participant.InviteStatus.IsAnswered() {
		return nil, domain.ErrInviteAlreadyAnswered
	}

	now := time.Now()
	participant.RespondedAt = &now
	participant.UpdatedAt = now

	if in.Accept {
		participant.InviteStatus = domain.InviteStatusAccepted
	} else {
		participant.InviteStatus = domain.InviteStatusRejected
		participant.Reason = in.Reason
	}

	if err := uc.participants.UpdateParticipant(ctx, participant); err != nil {
		return nil, err
	}

	// Best-effort: don't fail a recorded response just because the
	// notification didn't go out.
	if err := uc.notifier.Notify(ctx, domain.Notification{
		Type:        domain.NotificationInviteResponded,
		RecipientID: participant.InvitedBy,
		ActorID:     in.UserID,
		HangoutID:   in.HangoutID,
	}); err != nil {
		log.Printf("hangout: failed to notify invite response for hangout %s: %v", in.HangoutID, err)
	}

	return participant, nil
}
