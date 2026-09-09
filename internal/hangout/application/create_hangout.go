package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type CreateHangoutUseCase struct {
	hangouts     domain.HangoutRepository
	participants domain.ParticipantRepository
	tx           domain.Transactor
}

func NewCreateHangoutUseCase(hangouts domain.HangoutRepository, participants domain.ParticipantRepository, tx domain.Transactor) *CreateHangoutUseCase {
	return &CreateHangoutUseCase{hangouts: hangouts, participants: participants, tx: tx}
}

type CreateHangoutInput struct {
	OrganizerID uuid.UUID
	Title       string
	Description *string
	ScheduledAt *time.Time
}

// Execute creates the hangout row, then adds the organizer as an
// accepted participant. Both writes happen inside one transaction:
// if either fails, neither is kept.
func (uc *CreateHangoutUseCase) Execute(ctx context.Context, in CreateHangoutInput) (*domain.Hangout, error) {
	now := time.Now()

	hangout := &domain.Hangout{
		ID:          uuid.New(),
		OrganizerID: in.OrganizerID,
		Title:       in.Title,
		Description: in.Description,
		Status:      domain.HangoutStatusPlanned,
		ScheduledAt: in.ScheduledAt,
		Metadata:    map[string]any{},
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.hangouts.Create(ctx, hangout); err != nil {
			return err
		}

		organizer := &domain.Participant{
			ID:           uuid.New(),
			HangoutID:    hangout.ID,
			UserID:       in.OrganizerID,
			Role:         domain.ParticipantRoleOrganizer,
			InviteStatus: domain.InviteStatusAccepted,
			InvitedBy:    in.OrganizerID,
			InvitedAt:    now,
			RespondedAt:  &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		return uc.participants.AddParticipant(ctx, organizer)
	})
	if err != nil {
		return nil, err
	}

	return hangout, nil
}
