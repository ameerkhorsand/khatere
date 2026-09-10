package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InterestSource implements application.InterestSource by comparing
// a user's tagged interests (user_interests) against an activity's
// tagged interests (activity_interests, migration 0020).
type InterestSource struct {
	pool *pgxpool.Pool
}

func NewInterestSource(pool *pgxpool.Pool) *InterestSource {
	return &InterestSource{pool: pool}
}

// InterestMatch returns matched/total, where total is how many
// interests the activity is tagged with and matched is how many of
// those the user also has. An untagged activity returns 0.5, not 0
// — there's no evidence either way, so it shouldn't be buried under
// activities the user simply doesn't match.
func (s *InterestSource) InterestMatch(ctx context.Context, userID, activityID uuid.UUID) (float64, error) {
	var matched, total int
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE ui.interest_id IS NOT NULL) AS matched,
			COUNT(*) AS total
		FROM activity_interests ai
		LEFT JOIN user_interests ui
			ON ui.interest_id = ai.interest_id AND ui.user_id = $1
		WHERE ai.activity_id = $2
	`, userID, activityID).Scan(&matched, &total)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 0.5, nil
	}
	return float64(matched) / float64(total), nil
}
