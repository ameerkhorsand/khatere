package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bLorax/khatere-backend/internal/recommendation/domain"
)

// ActivityLookup implements application.ActivityLookup. It queries
// the activities table directly, the same way ActivityCandidateSource
// does, rather than importing internal/activity/domain — this
// package keeps its own narrow view of activity data instead of
// depending on that domain's types.
type ActivityLookup struct {
	pool *pgxpool.Pool
}

func NewActivityLookup(pool *pgxpool.Pool) *ActivityLookup {
	return &ActivityLookup{pool: pool}
}

func (l *ActivityLookup) ActivitySummaries(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID]domain.ActivitySummary, error) {
	out := make(map[uuid.UUID]domain.ActivitySummary, len(activityIDs))
	if len(activityIDs) == 0 {
		return out, nil
	}

	rows, err := l.pool.Query(ctx, `
		SELECT id, title, description, source_type
		FROM activities
		WHERE id = ANY($1) AND status = 'approved' AND deleted_at IS NULL
	`, activityIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.ActivitySummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Description, &s.SourceType); err != nil {
			return nil, err
		}
		out[s.ID] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
