package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------
// MeetupPin
// -----------------------------------------------------------------

// MeetupPin is the proposed time and place for a hangout. Exactly
// one pin exists per hangout at a time; a new proposal overwrites
// this row and bumps Version, which resets every confirmation.
type MeetupPin struct {
	ID          uuid.UUID
	HangoutID   uuid.UUID
	PlaceName   string
	Address     *string
	Latitude    float64
	Longitude   float64
	ScheduledAt *time.Time
	ProposedBy  uuid.UUID
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// -----------------------------------------------------------------
// PinConfirmation
// -----------------------------------------------------------------

type PinConfirmationStatus string

const (
	PinConfirmationPending   PinConfirmationStatus = "pending"
	PinConfirmationConfirmed PinConfirmationStatus = "confirmed"
	PinConfirmationDeclined  PinConfirmationStatus = "declined"
)

func (s PinConfirmationStatus) Valid() bool {
	switch s {
	case PinConfirmationPending, PinConfirmationConfirmed, PinConfirmationDeclined:
		return true
	default:
		return false
	}
}

type PinConfirmation struct {
	ID          uuid.UUID
	PinID       uuid.UUID
	UserID      uuid.UUID
	Status      PinConfirmationStatus
	ConfirmedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// -----------------------------------------------------------------
// PinSummary — the pin plus every participant's confirmation
// state, for a single read on the chat screen.
// -----------------------------------------------------------------

type PinSummary struct {
	Pin           MeetupPin
	Confirmations []PinConfirmation
}

// IsFullyConfirmed reports whether every listed confirmation has
// status Confirmed. An empty list is not fully confirmed.
func (s *PinSummary) IsFullyConfirmed() bool {
	if len(s.Confirmations) == 0 {
		return false
	}
	for _, c := range s.Confirmations {
		if c.Status != PinConfirmationConfirmed {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------
// Domain errors
// -----------------------------------------------------------------

var (
	ErrPinNotFound             = errors.New("meetup pin not found")
	ErrPinVersionConflict      = errors.New("meetup pin was modified by another request")
	ErrPinConfirmationNotFound = errors.New("pin confirmation not found for this user")
)
