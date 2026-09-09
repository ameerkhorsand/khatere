package application

import (
	"context"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type InviteParticipantsUseCase struct {
	hangouts     domain.HangoutRepository
	participants domain.ParticipantRepository
	notifier     domain.Notifier
}

func NewInviteParticipantsUseCase(hangouts domain.HangoutRepository, participants domain.ParticipantRepository, notifier domain.Notifier) *InviteParticipantsUseCase {
	return &InviteParticipantsUseCase{hangouts: hangouts, participants: participants, notifier: notifier}
}

type InviteParticipantsInput struct {
	HangoutID  uuid.UUID
	InviterID  uuid.UUID
	InviteeIDs []uuid.UUID
}

// InviteResult reports the outcome for one invitee. The use case
// invites everyone it can and reports failures per-user, instead of
// aborting the whole batch on the first error — the Invite to Hangout
// screen sends multiple invitees in one call.
type InviteResult struct {
	UserID      uuid.UUID
	Participant *domain.Participant
	Err         error
}

func (uc *InviteParticipantsUseCase) Execute(ctx context.Context, in InviteParticipantsInput) ([]InviteResult, error) {
	hangout, err := uc.hangouts.FindByID(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}
	if hangout.OrganizerID != in.InviterID {
		return nil, domain.ErrNotOrganizer
	}
	if hangout.Status.IsFinal() {
		return nil, domain.ErrHangoutIsFinal
	}

	activeCount, err := uc.participants.CountActive(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}

	results := make([]InviteResult, 0, len(in.InviteeIDs))
	now := time.Now()

	for _, inviteeID := range in.InviteeIDs {
		if activeCount >= domain.MaxParticipants {
			results = append(results, InviteResult{UserID: inviteeID, Err: domain.ErrParticipantLimit})
			continue
		}

		participant := &domain.Participant{
			ID:           uuid.New(),
			HangoutID:    in.HangoutID,
			UserID:       inviteeID,
			Role:         domain.ParticipantRoleParticipant,
			InviteStatus: domain.InviteStatusPending,
			InvitedBy:    in.InviterID,
			InvitedAt:    now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := uc.participants.AddParticipant(ctx, participant); err != nil {
			// Most likely domain.ErrAlreadyParticipant — record it
			// and keep going with the rest of the batch.
			results = append(results, InviteResult{UserID: inviteeID, Err: err})
			continue
		}

		activeCount++
		results = append(results, InviteResult{UserID: inviteeID, Participant: participant})

		// Best-effort: a notification failure must not undo an invite
		// that already succeeded, so we log and move on.
		if err := uc.notifier.Notify(ctx, domain.Notification{
			Type:        domain.NotificationHangoutInvite,
			RecipientID: inviteeID,
			ActorID:     in.InviterID,
			HangoutID:   in.HangoutID,
		}); err != nil {
			log.Printf("hangout: failed to notify invite for user %s: %v", inviteeID, err)
		}
	}

	return results, nil
}
