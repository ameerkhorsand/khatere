package application

import (
	"context"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type CreateHangoutUseCase struct {
	hangouts             domain.HangoutRepository
	participants         domain.ParticipantRepository
	tx                   domain.Transactor
	suggestionAcceptance domain.SuggestionAcceptanceRecorder
}

func NewCreateHangoutUseCase(hangouts domain.HangoutRepository, participants domain.ParticipantRepository, tx domain.Transactor, suggestionAcceptance domain.SuggestionAcceptanceRecorder) *CreateHangoutUseCase {
	return &CreateHangoutUseCase{hangouts: hangouts, participants: participants, tx: tx, suggestionAcceptance: suggestionAcceptance}
}

type CreateHangoutInput struct {
	OrganizerID uuid.UUID
	ActivityID  *uuid.UUID // nil for a hangout with no linked activity
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
		ActivityID:  in.ActivityID,
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

	// Best-effort, same reasoning as the notifications in
	// CancelHangoutUseCase: the hangout is already created, so a
	// failure here is logged, not returned. Only recorded when this
	// hangout is actually organized around a tagged activity — a
	// hangout with no ActivityID has nothing for the recommendation
	// engine to reinforce.
	if in.ActivityID != nil {
		if err := uc.suggestionAcceptance.RecordAccepted(ctx, in.OrganizerID, *in.ActivityID); err != nil {
			log.Printf("hangout: failed to record suggestion acceptance for user %s, activity %s: %v", in.OrganizerID, *in.ActivityID, err)
		}
	}

	return hangout, nil
}
