package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/recommendation/domain"
	"github.com/google/uuid"
)

// The four interfaces below are the seams between this domain and
// the rest of the app. Each is deliberately narrow — one method,
// one number — so a Postgres adapter can satisfy it by querying
// whatever tables it needs (interests, hangouts, ratings, comments)
// without this package importing those domains directly. This
// mirrors AttendanceChecker in internal/rating/application.

// InterestSource resolves how well an activity matches a user's
// stated/inferred interests. Returns a value in [0, 1]; 0 means no
// overlap, 1 means a perfect match on the interest lookup table.
type InterestSource interface {
	InterestMatch(ctx context.Context, userID, activityID uuid.UUID) (float64, error)
}

// HistorySource resolves how often the user has engaged with this
// activity, or activities like it, before. Returns a value in
// [0, 1] derived from past acceptance/attendance rate.
type HistorySource interface {
	HistoryAffinity(ctx context.Context, userID, activityID uuid.UUID) (float64, error)
}

// EngagementSource resolves the user's general in-app activity
// level, independent of any single activity. Returns a value in
// [0, 1].
type EngagementSource interface {
	Engagement(ctx context.Context, userID uuid.UUID) (float64, error)
}

// QualitySource resolves an activity's own reputation — its rating
// average and comment-vote quality, independent of any one user.
// Returns a value in [0, 1].
type QualitySource interface {
	Quality(ctx context.Context, activityID uuid.UUID) (float64, error)
}

// ActivityLookup resolves the display data for a batch of activity
// IDs in one call, so the /recommendations response can embed each
// suggestion's activity without a separate request per suggestion.
// Only approved, non-deleted activities are expected back — an ID
// with no match (deleted or rejected since it was cached) is simply
// left out of the returned map, not an error.
type ActivityLookup interface {
	ActivitySummaries(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID]domain.ActivitySummary, error)
}
