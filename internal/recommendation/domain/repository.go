package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RecommendationRepository persists and reads the ranked-score
// cache (user_recommendation_cache, migration TBD in Step 5). This
// interface only covers this domain's own storage — ports for
// pulling raw signal data out of other domains (interests, ratings,
// hangouts, comments) live in the application layer, next to the
// use case that needs them, not here.
type RecommendationRepository interface {
	// ReplaceForUser overwrites the cached scores for one user with
	// a freshly computed set, in a single transaction. A full
	// replace (not an upsert-per-row) keeps stale activities from
	// lingering in a user's feed after their score drops out.
	ReplaceForUser(ctx context.Context, userID uuid.UUID, scores []Score) error

	// TopForUser returns a user's cached scores ordered by Total
	// descending, capped at limit. Returns an empty slice, not an
	// error, when the cache is empty — callers decide whether that
	// means "compute live" or "show nothing yet".
	TopForUser(ctx context.Context, userID uuid.UUID, limit int) ([]Score, error)

	// StaleUserIDs returns every distinct user ID whose newest
	// cached score is older than cutoff — i.e. every user this
	// background refresh worker (Step 5) should recompute this run.
	// A user with no cache row at all is not returned here; they're
	// covered by GetSuggestionsUseCase's existing lazy cache-miss
	// path instead, the first time they call /recommendations.
	StaleUserIDs(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error)
}

// SuggestionStatsRepository persists and reads per-user,
// per-activity suggestion/acceptance counters
// (user_suggestion_stats, migration 0021). This is this domain's
// own storage — same reasoning as RecommendationRepository above.
// The write path that increments TimesAccepted from outside this
// domain (e.g. when a hangout is created) goes through a narrow
// port defined in application/, not through this interface
// directly — see the handoff doc's cross-domain rule.
type SuggestionStatsRepository interface {
	// RecordSuggested increments TimesSuggested and sets
	// LastSuggestedAt for each of the given activity IDs, for one
	// user, creating each row if it doesn't exist yet. This is an
	// upsert-per-row, not a ReplaceAll — these are running counters,
	// not a set-of-tags like activity_interests.
	RecordSuggested(ctx context.Context, userID uuid.UUID, activityIDs []uuid.UUID) error

	// RecordAccepted increments TimesAccepted for one
	// (userID, activityID) pair, creating the row if it doesn't
	// exist yet.
	RecordAccepted(ctx context.Context, userID, activityID uuid.UUID) error

	// Get returns the stats row for one (userID, activityID) pair.
	// Returns the zero value (TimesSuggested=0, TimesAccepted=0),
	// not an error, when no row exists — callers rely on
	// SuggestionStats.AcceptanceRate()'s neutral-0.5 fallback for
	// that case.
	Get(ctx context.Context, userID, activityID uuid.UUID) (SuggestionStats, error)
}
