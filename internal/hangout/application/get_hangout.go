package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type GetHangoutUseCase struct {
	hangouts     domain.HangoutRepository
	participants domain.ParticipantRepository
}

func NewGetHangoutUseCase(hangouts domain.HangoutRepository, participants domain.ParticipantRepository) *GetHangoutUseCase {
	return &GetHangoutUseCase{hangouts: hangouts, participants: participants}
}

type GetHangoutInput struct {
	HangoutID   uuid.UUID
	RequesterID uuid.UUID
}

// HangoutDetail bundles a hangout with its participant list — the
// Hangout Chat/Planning screen needs both to render who's coming and
// who's still deciding.
type HangoutDetail struct {
	Hangout      domain.Hangout
	Participants []domain.Participant
}

// Execute only returns the hangout to someone already listed as a
// participant (organizer, or invited — any invite status). This
// keeps hangout details private from anyone not invited.
func (uc *GetHangoutUseCase) Execute(ctx context.Context, in GetHangoutInput) (*HangoutDetail, error) {
	hangout, err := uc.hangouts.FindByID(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}

	if _, err := uc.participants.FindParticipant(ctx, in.HangoutID, in.RequesterID); err != nil {
		return nil, domain.ErrNotInvited
	}

	participants, err := uc.participants.ListParticipants(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}

	return &HangoutDetail{Hangout: *hangout, Participants: participants}, nil
}
