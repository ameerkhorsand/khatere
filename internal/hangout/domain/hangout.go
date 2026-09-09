package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MaxParticipants is the hard cap on hangout size (1 organizer + up to
// 99 invitees = 100 total). Enforced in the application layer.
const MaxParticipants = 100

// -----------------------------------------------------------------
// Hangout
// -----------------------------------------------------------------

type HangoutStatus string

const (
	HangoutStatusPlanned   HangoutStatus = "planned"
	HangoutStatusOngoing   HangoutStatus = "ongoing"
	HangoutStatusCompleted HangoutStatus = "completed"
	HangoutStatusCancelled HangoutStatus = "cancelled"
)

func (s HangoutStatus) Valid() bool {
	switch s {
	case HangoutStatusPlanned, HangoutStatusOngoing, HangoutStatusCompleted, HangoutStatusCancelled:
		return true
	default:
		return false
	}
}

// IsFinal reports whether a hangout in this status can no longer change.
func (s HangoutStatus) IsFinal() bool {
	return s == HangoutStatusCompleted || s == HangoutStatusCancelled
}

type Hangout struct {
	ID             uuid.UUID
	ActivityID     *uuid.UUID // nullable: a user-organized hangout may have no linked activity
	OrganizerID    uuid.UUID
	Title          string
	Description    *string
	Status         HangoutStatus
	ScheduledAt    *time.Time
	ScheduledEndAt *time.Time // nullable: only known hangouts with a fixed end time
	Metadata       map[string]any
	Version        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

func (h *Hangout) IsDeleted() bool {
	return h.DeletedAt != nil
}

// -----------------------------------------------------------------
// Participant (also carries the invite: see repository.go note)
// -----------------------------------------------------------------

type ParticipantRole string

const (
	ParticipantRoleOrganizer   ParticipantRole = "organizer"
	ParticipantRoleParticipant ParticipantRole = "participant"
)

type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusRejected InviteStatus = "rejected"
)

func (s InviteStatus) Valid() bool {
	switch s {
	case InviteStatusPending, InviteStatusAccepted, InviteStatusRejected:
		return true
	default:
		return false
	}
}

// IsAnswered reports whether the invitee has already responded.
func (s InviteStatus) IsAnswered() bool {
	return s == InviteStatusAccepted || s == InviteStatusRejected
}

type Participant struct {
	ID           uuid.UUID
	HangoutID    uuid.UUID
	UserID       uuid.UUID
	Role         ParticipantRole
	InviteStatus InviteStatus
	Reason       *string
	InvitedBy    uuid.UUID
	InvitedAt    time.Time
	RespondedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsActive reports whether this participant currently counts toward
// the hangout (accepted, or still deciding).
func (p *Participant) IsActive() bool {
	return p.InviteStatus == InviteStatusAccepted || p.InviteStatus == InviteStatusPending
}

// -----------------------------------------------------------------
// Message
// -----------------------------------------------------------------

type Message struct {
	ID        uuid.UUID
	HangoutID uuid.UUID
	SenderID  uuid.UUID
	Content   string
	CreatedAt time.Time
	DeletedAt *time.Time
}

func (m *Message) IsDeleted() bool {
	return m.DeletedAt != nil
}

// -----------------------------------------------------------------
// Domain errors
// -----------------------------------------------------------------

var (
	ErrHangoutNotFound        = errors.New("hangout not found")
	ErrInvalidHangoutStatus   = errors.New("invalid hangout status")
	ErrHangoutIsFinal         = errors.New("hangout is completed or cancelled and cannot be changed")
	ErrNotOrganizer           = errors.New("only the organizer can perform this action")
	ErrVersionConflict        = errors.New("hangout was modified by another request")
	ErrParticipantLimit       = errors.New("hangout cannot have more than 100 participants")
	ErrParticipantNotFound    = errors.New("participant not found")
	ErrAlreadyParticipant     = errors.New("user is already invited to this hangout")
	ErrInviteAlreadyAnswered  = errors.New("invite has already been accepted or rejected")
	ErrNotInvited             = errors.New("user is not invited to this hangout")
	ErrNotAcceptedParticipant = errors.New("only accepted participants can post messages")
)
