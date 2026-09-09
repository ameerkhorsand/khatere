package application

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

// ProposeMeetupPinUseCase covers both the first pin proposal for a
// hangout and every later change to it. A change always resets every
// participant's confirmation back to pending — there is no deadline,
// so a pin simply stays open until everyone re-confirms.
type ProposeMeetupPinUseCase struct {
	hangouts      domain.HangoutRepository
	participants  domain.ParticipantRepository
	pins          domain.MeetupPinRepository
	confirmations domain.PinConfirmationRepository
	notifier      domain.Notifier
	tx            domain.Transactor
}

func NewProposeMeetupPinUseCase(
	hangouts domain.HangoutRepository,
	participants domain.ParticipantRepository,
	pins domain.MeetupPinRepository,
	confirmations domain.PinConfirmationRepository,
	notifier domain.Notifier,
	tx domain.Transactor,
) *ProposeMeetupPinUseCase {
	return &ProposeMeetupPinUseCase{
		hangouts:      hangouts,
		participants:  participants,
		pins:          pins,
		confirmations: confirmations,
		notifier:      notifier,
		tx:            tx,
	}
}

type ProposeMeetupPinInput struct {
	HangoutID   uuid.UUID
	ProposerID  uuid.UUID
	PlaceName   string
	Address     *string
	Latitude    float64
	Longitude   float64
	ScheduledAt *time.Time
}

func (uc *ProposeMeetupPinUseCase) Execute(ctx context.Context, in ProposeMeetupPinInput) (*domain.MeetupPin, error) {
	hangout, err := uc.hangouts.FindByID(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}
	if hangout.OrganizerID != in.ProposerID {
		return nil, domain.ErrNotOrganizer
	}
	if hangout.Status.IsFinal() {
		return nil, domain.ErrHangoutIsFinal
	}

	allParticipants, err := uc.participants.ListParticipants(ctx, in.HangoutID)
	if err != nil {
		return nil, err
	}

	// Only accepted participants need to confirm a place and time.
	// A pending invitee has not joined the hangout yet; a declined
	// invitee is no longer part of it.
	activeUserIDs := make([]uuid.UUID, 0, len(allParticipants))
	for _, p := range allParticipants {
		if p.InviteStatus == domain.InviteStatusAccepted {
			activeUserIDs = append(activeUserIDs, p.UserID)
		}
	}

	now := time.Now()
	existing, err := uc.pins.FindByHangoutID(ctx, in.HangoutID)
	isFirstProposal := errors.Is(err, domain.ErrPinNotFound)
	if err != nil && !isFirstProposal {
		return nil, err
	}

	pin := &domain.MeetupPin{
		HangoutID:   in.HangoutID,
		PlaceName:   in.PlaceName,
		Address:     in.Address,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
		ScheduledAt: in.ScheduledAt,
		ProposedBy:  in.ProposerID,
		UpdatedAt:   now,
	}

	err = uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if isFirstProposal {
			pin.ID = uuid.New()
			pin.Version = 1
			pin.CreatedAt = now
			if err := uc.pins.Create(ctx, pin); err != nil {
				return err
			}
		} else {
			pin.ID = existing.ID
			pin.Version = existing.Version
			pin.CreatedAt = existing.CreatedAt
			if err := uc.pins.Update(ctx, pin); err != nil {
				return err
			}
			// The Update WHERE clause matched on the pre-update
			// version. The set_updated_at_and_bump_version trigger
			// then bumped the stored row to version+1 — reflect
			// that here so the response we return matches what is
			// actually in the database, instead of the stale
			// pre-update value.
			pin.Version = existing.Version + 1
		}

		return uc.confirmations.ResetForPin(ctx, pin.ID, activeUserIDs)
	})
	if err != nil {
		return nil, err
	}

	notificationType := domain.NotificationMeetupPinProposed
	if !isFirstProposal {
		notificationType = domain.NotificationMeetupPinChanged
	}

	// Best-effort: the pin is already saved, so a notification
	// failure here must not undo it.
	for _, userID := range activeUserIDs {
		if userID == in.ProposerID {
			continue
		}
		if err := uc.notifier.Notify(ctx, domain.Notification{
			Type:        notificationType,
			RecipientID: userID,
			ActorID:     in.ProposerID,
			HangoutID:   in.HangoutID,
		}); err != nil {
			log.Printf("hangout: failed to notify meetup pin change for user %s: %v", userID, err)
		}
	}

	return pin, nil
}
