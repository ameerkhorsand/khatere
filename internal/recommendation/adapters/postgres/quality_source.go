package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QualitySource implements application.QualitySource by blending two
// existing reputation signals for an activity: its rating average
// and how positively its approved comments have been voted. Both are
// weighted equally — there's no evidence yet that one predicts
// enjoyment better than the other, so 50/50 is the honest starting
// point until real usage data says otherwise.
type QualitySource struct {
	pool *pgxpool.Pool
}

func NewQualitySource(pool *pgxpool.Pool) *QualitySource {
	return &QualitySource{pool: pool}
}

// Quality averages the rating component (AVG(score)/5) with the
// comment component (upvotes / total votes, across approved
// comments only). Either component defaults to 0.5 — neutral — when
// there's no data yet, same reasoning as the other three sources.
func (s *QualitySource) Quality(ctx context.Context, activityID uuid.UUID) (float64, error) {
	var ratingAvg *float64
	var ratingCount int
	var upvotes, totalVotes int

	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT AVG(score) FROM ratings WHERE activity_id = $1) AS rating_avg,
			(SELECT COUNT(*) FROM ratings WHERE activity_id = $1) AS rating_count,
			(SELECT COUNT(*) FROM comment_votes cv
			 JOIN comments c ON c.id = cv.comment_id
			 WHERE c.activity_id = $1 AND c.status = 'approved' AND cv.value = 1) AS upvotes,
			(SELECT COUNT(*) FROM comment_votes cv
			 JOIN comments c ON c.id = cv.comment_id
			 WHERE c.activity_id = $1 AND c.status = 'approved') AS total_votes
	`, activityID).Scan(&ratingAvg, &ratingCount, &upvotes, &totalVotes)
	if err != nil {
		return 0, err
	}

	ratingComponent := 0.5
	if ratingCount > 0 && ratingAvg != nil {
		ratingComponent = *ratingAvg / 5.0
	}

	commentComponent := 0.5
	if totalVotes > 0 {
		commentComponent = float64(upvotes) / float64(totalVotes)
	}

	return (ratingComponent + commentComponent) / 2, nil
}
