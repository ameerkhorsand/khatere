package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserBadgeRepository interface {
	// Create inserts a new award. Fails on the
	// UNIQUE(user_id, badge_id) constraint (migration 0018) if
	// this user already has this badge — callers should check
	// FindByUserAndBadge first, same pattern as attendance's
	// Create.
	Create(ctx context.Context, ub *UserBadge) error

	FindByUserAndBadge(ctx context.Context, userID, badgeID uuid.UUID) (*UserBadge, error)

	// ListByUser backs the Profile "Enjoyed" section (Phase 7
	// Step 6), most recently awarded first.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]UserBadge, error)
}
