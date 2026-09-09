package application

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

// UpdateHangoutStatusUseCase covers the organizer-driven transitions
// planned -> ongoing and ongoing -> completed. Cancel has its own use
// case (CancelHangoutUseCase) because it needs different rules — it
// can be called from any non-final status, not just the next step in
// sequence.
type UpdateHangoutStatusUseCase struct {
	hangouts       domain.HangoutRepository
	archiveCreator domain.ArchiveCreator
}

func NewUpdateHangoutStatusUseCase(hangouts domain.HangoutRepository, archiveCreator domain.ArchiveCreator) *UpdateHangoutStatusUseCase {
	return &UpdateHangoutStatusUseCase{hangouts: hangouts, archiveCreator: archiveCreator}
}

type UpdateHangoutStatusInput struct {
	HangoutID   uuid.UUID
	RequesterID uuid.UUID
	NewStatus   domain.HangoutStatus
}

// validNextStatus lists the one allowed forward step from each
// non-final status.
var validNextStatus = map[domain.HangoutStatus]domain.HangoutStatus{
	domain.HangoutStatusPlanned: domain.HangoutStatusOngoing,
	domain.HangoutStatusOngoing: domain.HangoutStatusCompleted,
}

func (uc *UpdateHangoutStatusUseCase) Execute(ctx context.Context, in UpdateHangoutStatusInput) (*domain.Hangout, error) {
	if !in.NewStatus.Valid() {
		return nil, domain.ErrInvalidHangoutStatus
	}

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
	if validNextStatus[hangout.Status] != in.NewStatus {
		return nil, domain.ErrInvalidHangoutStatus
	}

	hangout.Status = in.NewStatus

	if err := uc.hangouts.Update(ctx, hangout); err != nil {
		return nil, err
	}

	if in.NewStatus == domain.HangoutStatusCompleted {
		if err := uc.archiveCreator.CreateArchive(ctx, in.HangoutID); err != nil {
			log.Printf("hangout: failed to create archive for completed hangout %s: %v", in.HangoutID, err)
		}
	}

	return hangout, nil
}
