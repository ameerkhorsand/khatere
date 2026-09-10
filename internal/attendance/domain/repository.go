package domain

import (
	"context"

	"github.com/google/uuid"
)

type AttendanceVerificationRepository interface {
	// Create inserts a new verification. Fails on the
	// UNIQUE(activity_id, user_id) constraint (migration 0018) if
	// one already exists — callers should check
	// FindByActivityAndUser first, same pattern as rating's Upsert.
	Create(ctx context.Context, v *AttendanceVerification) error

	FindByActivityAndUser(ctx context.Context, activityID, userID uuid.UUID) (*AttendanceVerification, error)
}
