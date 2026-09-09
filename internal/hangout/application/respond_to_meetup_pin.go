package application

import (
	"context"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

// RespondToMeetupPinUseCase lets one participant set their own
// confirmation status against the hangout's current pin.
type RespondToMeetupPinUseCase struct {
	hangouts      domain.HangoutRepository
	pins          domain.MeetupPinRepository
	confirmations domain.PinConfirmationRepository
	notifier      domain.Notifier
}

func NewRespondToMeetupPinUseCase(
	hangouts domain.HangoutRepository,
	pins domain.MeetupPinRepository,
	confirmations domain.PinConfirmationRepository,
	notifier domain.Notifier,
) *RespondToMeetupPinUseCase {
	return &RespondToMeetupPinUseCase{hangouts: hangouts, pins: pins, confirmations: confirmations, notifier: notifier}
}

type RespondToMeetupPinInput struct {
	HangoutID uuid.UUID
	UserID    uuid.UUID
	Confirm   bool
}

func (uc *RespondToMeetupPinUseCase) Execute(ctx context.Context, in RespondToMeetupPinInput) (*domain.PinConfirmation, error) {
	hangout, err := uc.hangouts.FindByID(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}
	if hangout.Status.IsFinal() {
		return nil, domain.ErrHangoutIsFinal
	}

	pin, err := uc.pins.FindByHangoutID(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}

	confirmation, err := uc.confirmations.FindConfirmation(ctx, pin.ID, in.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	confirmation.UpdatedAt = now
	if in.Confirm {
		confirmation.Status = domain.PinConfirmationConfirmed
		confirmation.ConfirmedAt = &now
	} else {
		confirmation.Status = domain.PinConfirmationDeclined
		confirmation.ConfirmedAt = nil
	}

	if err := uc.confirmations.UpdateConfirmation(ctx, confirmation); err != nil {
		return nil, err
	}

	if in.Confirm {
		// Best-effort: a recorded confirmation must stand even if
		// the organizer never hears about it.
		if err := uc.notifier.Notify(ctx, domain.Notification{
			Type:        domain.NotificationMeetupPinConfirmed,
			RecipientID: pin.ProposedBy,
			ActorID:     in.UserID,
			HangoutID:   in.HangoutID,
		}); err != nil {
			log.Printf("hangout: failed to notify meetup pin confirmation for hangout %s: %v", in.HangoutID, err)
		}
	}

	return confirmation, nil
}
