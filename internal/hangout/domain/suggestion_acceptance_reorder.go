package domain

import (
	"context"

	"github.com/google/uuid"
)

// SuggestionAcceptanceRecorder lets the hangout module tell the
// recommendation module that a user has organized a hangout around
// a specific activity, without importing the recommendation module
// directly — same small-port pattern as ArchiveCreator (see
// archive_creator.go) and Notifier (see notification.go).
type SuggestionAcceptanceRecorder interface {
	RecordAccepted(ctx context.Context, userID, activityID uuid.UUID) error
}
