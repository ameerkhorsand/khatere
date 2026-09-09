package domain

import (
	"context"

	"github.com/google/uuid"
)

type ListFilter struct {
	Status     *ActivityStatus // nil = any status
	SourceType *SourceType     // nil = any source
}

type ActivityRepository interface {
	Create(ctx context.Context, activity *Activity) error
	FindByID(ctx context.Context, id uuid.UUID) (*Activity, error)
	// List returns non-deleted activities matching the filter, most recent first.
	List(ctx context.Context, filter ListFilter) ([]Activity, error)
	// ListPendingQueue returns pending, non-deleted activities, oldest first
	// (fair queue ordering for moderators).
	ListPendingQueue(ctx context.Context) ([]Activity, error)
	// Update performs an optimistic-lock update: checks activity.Version
	// against the stored row, returns ErrVersionConflict on mismatch.
	Update(ctx context.Context, activity *Activity) error
}
