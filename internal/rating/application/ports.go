package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/google/uuid"
)

// SummaryCache is a fast, disposable read-through cache in front of
// RatingRepository.Summarize. It is never the source of truth —
// Postgres is — so a cache miss or a cache being entirely empty
// must always be a safe, correct fallback, never an error.
type SummaryCache interface {
	// Get returns found=false, with a nil error, on a cache miss.
	// A real Redis error is still returned as err, so the caller
	// can decide to fall back to Postgres rather than fail the
	// request outright.
	Get(ctx context.Context, activityID uuid.UUID) (summary domain.Summary, found bool, err error)

	Set(ctx context.Context, activityID uuid.UUID, summary domain.Summary) error

	// Invalidate deletes the cached entry. A no-op, not an error,
	// if nothing was cached for this activity yet.
	Invalidate(ctx context.Context, activityID uuid.UUID) error
}
