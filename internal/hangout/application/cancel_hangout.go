package application

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type CancelHangoutUseCase struct {
	hangouts       domain.HangoutRepository
	participants   domain.ParticipantRepository
	notifier       domain.Notifier
	archiveCreator domain.ArchiveCreator
}

func NewCancelHangoutUseCase(hangouts domain.HangoutRepository, participants domain.ParticipantRepository, notifier domain.Notifier, archiveCreator domain.ArchiveCreator) *CancelHangoutUseCase {
	return &CancelHangoutUseCase{hangouts: hangouts, participants: participants, notifier: notifier, archiveCreator: archiveCreator}
}

type CancelHangoutInput struct {
	HangoutID   uuid.UUID
	RequesterID uuid.UUID
}

func (uc *CancelHangoutUseCase) Execute(ctx context.Context, in CancelHangoutInput) (*domain.Hangout, error) {
	hangout, err := uc.hangouts.FindByID(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}
	if hangout.OrganizerID != in.RequesterID {
		return nil, domain.ErrNotOrganizer
	}
	if hangout.Status.IsFinal() {
		return nil, domain.ErrHangoutIsFinal
	}

	hangout.Status = domain.HangoutStatusCancelled

	if err := uc.hangouts.Update(ctx, hangout); err != nil {
		return nil, err
	}

	// >>> NEW — add this block here <
	// Best-effort, same as the notifications below: the cancellation
	// is already recorded, so a failure here is logged, not returned.
	if err := uc.archiveCreator.CreateArchive(ctx, in.HangoutID); err != nil {
		log.Printf("hangout: failed to create archive for cancelled hangout %s: %v", in.HangoutID, err)
	}
	// >>> end new block <

	// Best-effort: the cancellation is already recorded, so a
	// notification failure here is logged, not returned.
	participants, err := uc.participants.ListParticipants(ctx, in.HangoutID)
	if err != nil {
		log.Printf("hangout: failed to list participants to notify for cancelled hangout %s: %v", in.HangoutID, err)
		return hangout, nil
	}
	for _, p := range participants {
		if p.UserID == in.RequesterID {
			continue // the organizer doesn't need to be told they cancelled it
		}
		if err := uc.notifier.Notify(ctx, domain.Notification{
			Type:        domain.NotificationHangoutCancelled,
			RecipientID: p.UserID,
			ActorID:     in.RequesterID,
			HangoutID:   in.HangoutID,
		}); err != nil {
			log.Printf("hangout: failed to notify cancellation for user %s: %v", p.UserID, err)
		}
	}

	return hangout, nil
}
