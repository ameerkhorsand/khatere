package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByAccountID(ctx context.Context, accountID uuid.UUID) (*User, error)
	// Update performs an optimistic-lock update: it checks user.Version
	// against the stored row and returns ErrVersionConflict on mismatch.
	Update(ctx context.Context, user *User) error
}

type InterestRepository interface {
	// ListActive returns the full catalog of active interests, for onboarding UI.
	ListActive(ctx context.Context) ([]Interest, error)
	// ReplaceAll deletes the user's existing interest selections and inserts
	// the given interest IDs, as a single transaction.
	ReplaceAll(ctx context.Context, userID uuid.UUID, interestIDs []uuid.UUID) error
	// FindByUserID returns the full Interest records the user has selected.
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]Interest, error)
}
