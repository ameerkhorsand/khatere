package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

// GetMeetupPinUseCase reads the current pin for a hangout, along
// with every participant's confirmation state. This is the single
// call the pinned banner on the Hangout Chat screen needs.
type GetMeetupPinUseCase struct {
	participants  domain.ParticipantRepository
	pins          domain.MeetupPinRepository
	confirmations domain.PinConfirmationRepository
}

func NewGetMeetupPinUseCase(
	participants domain.ParticipantRepository,
	pins domain.MeetupPinRepository,
	confirmations domain.PinConfirmationRepository,
) *GetMeetupPinUseCase {
	return &GetMeetupPinUseCase{participants: participants, pins: pins, confirmations: confirmations}
}

type GetMeetupPinInput struct {
	HangoutID   uuid.UUID
	RequesterID uuid.UUID
}

func (uc *GetMeetupPinUseCase) Execute(ctx context.Context, in GetMeetupPinInput) (*domain.PinSummary, error) {
	// Only a participant of the hangout may view its pin.
	if _, err := uc.participants.FindParticipant(ctx, in.HangoutID, in.RequesterID); err != nil {
		return nil, err
	}

	pin, err := uc.pins.FindByHangoutID(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}

	confirmations, err := uc.confirmations.ListByPin(ctx, pin.ID)
	if err != nil {
		return nil, err
	}

	return &domain.PinSummary{Pin: *pin, Confirmations: confirmations}, nil
}
