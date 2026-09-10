package domain

import (
	"context"

	"github.com/google/uuid"
)

type BadgeRepository interface {
	// Create inserts a new badge. Fails on the UNIQUE(activity_id)
	// constraint (migration 0018) if the activity already has one —
	// callers should check FindByActivityID first, same pattern as
	// qrcode's Create.
	Create(ctx context.Context, b *Badge) error

	FindByActivityID(ctx context.Context, activityID uuid.UUID) (*Badge, error)

	// FindByID backs the badge detail screen (Phase 7 Step 6) and
	// the userbadge domain's award step (Phase 7 Step 5).
	FindByID(ctx context.Context, id uuid.UUID) (*Badge, error)
}
