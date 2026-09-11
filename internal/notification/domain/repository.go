package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists and reads notifications (the notifications
// table, migration in the next step). The consumer worker (step 8)
// is the only writer via Create — nothing else should insert a row
// directly, since that would bypass the Kafka pipeline this phase
// exists to build.
type Repository interface {
	// Create inserts one notification. Called only by the Kafka
	// consumer worker, once per event it reads off the topic.
	Create(ctx context.Context, n *Notification) error

	// ListForRecipient returns one user's notifications, newest
	// first, capped at limit. Returns an empty slice, not an
	// error, when there are none yet.
	ListForRecipient(ctx context.Context, recipientID uuid.UUID, limit int) ([]Notification, error)

	// MarkRead sets ReadAt on one notification, only if it belongs
	// to recipientID — this check happens in the same query so a
	// caller can never mark someone else's notification read.
	MarkRead(ctx context.Context, notificationID, recipientID uuid.UUID) error
}
