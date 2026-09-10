// Package suggestionacceptance is the one place the hangout module
// reaches across into the recommendation module — the same shape as
// internal/hangout/adapters/archive, which reaches into the archive
// module for the same reason.
package suggestionacceptance

import (
	"context"

	recommendationDomain "github.com/bLorax/khatere-backend/internal/recommendation/domain"
	"github.com/google/uuid"
)

// Recorder implements hangout/domain.SuggestionAcceptanceRecorder by
// wrapping the recommendation domain's own repository port. There's
// no extra business logic to add on top (unlike archive's Creator,
// which wraps a full use case) — RecordAccepted is already a single,
// narrow write, so wrapping the repository directly avoids an empty
// pass-through use case.
type Recorder struct {
	stats recommendationDomain.SuggestionStatsRepository
}

func New(stats recommendationDomain.SuggestionStatsRepository) *Recorder {
	return &Recorder{stats: stats}
}

func (r *Recorder) RecordAccepted(ctx context.Context, userID, activityID uuid.UUID) error {
	return r.stats.RecordAccepted(ctx, userID, activityID)
}
