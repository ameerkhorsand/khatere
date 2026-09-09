package domain

import (
	"context"

	"github.com/google/uuid"
)

// ArchiveCreator lets the hangout module trigger archive creation
// once a hangout resolves (Completed or Cancelled), without
// importing the archive module directly — same small-port pattern
// as Notifier (see notification.go).
type ArchiveCreator interface {
	CreateArchive(ctx context.Context, hangoutID uuid.UUID) error
}
