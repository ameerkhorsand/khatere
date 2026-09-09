package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListFilter narrows ListByUser. Status nil = any status. The four
// frontend tabs (Upcoming / Ongoing / Completed / Didn't Happen) map
// to Planned / Ongoing / Completed / Cancelled.
type ListFilter struct {
	Status *HangoutStatus
}

// MessageFilter pages through a hangout's chat, oldest-fetched-last.
// Before nil = start from the most recent message.
type MessageFilter struct {
	Before *time.Time
	Limit  int
}

type HangoutRepository interface {
	// Create inserts a new hangout row. It does not add any
	// participants; the caller adds the organizer as a participant
	// separately, in the same use case.
	Create(ctx context.Context, hangout *Hangout) error

	FindByID(ctx context.Context, id uuid.UUID) (*Hangout, error)

	// ListByUser returns hangouts where userID is an active
	// participant (organizer or accepted/pending invitee), most
	// recent scheduled_at first.
	ListByUser(ctx context.Context, userID uuid.UUID, filter ListFilter) ([]Hangout, error)

	// Update performs an optimistic-lock update: checks
	// hangout.Version against the stored row, returns
	// ErrVersionConflict on mismatch.
	Update(ctx context.Context, hangout *Hangout) error
}

type ParticipantRepository interface {
	// AddParticipant inserts one invite/participant row. Returns
	// ErrAlreadyParticipant if the user already has a row for this
	// hangout.
	AddParticipant(ctx context.Context, participant *Participant) error

	FindParticipant(ctx context.Context, hangoutID, userID uuid.UUID) (*Participant, error)

	ListParticipants(ctx context.Context, hangoutID uuid.UUID) ([]Participant, error)

	// CountActive returns the number of participants with status
	// pending or accepted. Used to enforce MaxParticipants before an
	// invite is sent.
	CountActive(ctx context.Context, hangoutID uuid.UUID) (int, error)

	// UpdateParticipant writes back an invite response (status,
	// reason, responded_at). Returns ErrInviteAlreadyAnswered if the
	// stored row is no longer pending.
	UpdateParticipant(ctx context.Context, participant *Participant) error
}

// Transactor lets a use case group several repository writes into one
// atomic database transaction, without the use case or the domain
// package knowing anything about the underlying database. fn receives
// a context carrying the active transaction; every repository call
// made with that context joins the same transaction.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, message *Message) error

	// ListMessages returns non-deleted messages for a hangout,
	// newest first, matching filter.
	ListMessages(ctx context.Context, hangoutID uuid.UUID, filter MessageFilter) ([]Message, error)
}
