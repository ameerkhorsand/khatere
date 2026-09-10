package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HistorySource implements application.HistorySource by looking at
// how the user's own past hangouts for this exact activity turned
// out — completed versus cancelled. Cross-activity, category-level
// recurrence (e.g. "this user usually accepts hiking") is a
// separate mechanism, built in Step 4 as user_suggestion_stats, not
// this port.
type HistorySource struct {
	pool *pgxpool.Pool
}

func NewHistorySource(pool *pgxpool.Pool) *HistorySource {
	return &HistorySource{pool: pool}
}

// HistoryAffinity counts the user's own hangouts for this activity
// as either organizer or an accepted participant, and returns
// completed / (completed + cancelled). With no history at all for
// this activity, it returns 0.5 — neutral, same reasoning as
// InterestSource for an untagged activity.
func (s *HistorySource) HistoryAffinity(ctx context.Context, userID, activityID uuid.UUID) (float64, error) {
	var completed, cancelled int
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE h.status = 'completed') AS completed,
			COUNT(*) FILTER (WHERE h.status = 'cancelled') AS cancelled
		FROM hangouts h
		WHERE h.activity_id = $2
		  AND h.deleted_at IS NULL
		  AND (
		      h.organizer_id = $1
		      OR EXISTS (
		          SELECT 1 FROM hangout_participants hp
		          WHERE hp.hangout_id = h.id
		            AND hp.user_id = $1
		            AND hp.invite_status = 'accepted'
		      )
		  )
	`, userID, activityID).Scan(&completed, &cancelled)
	if err != nil {
		return 0, err
	}

	total := completed + cancelled
	if total == 0 {
		return 0.5, nil
	}
	return float64(completed) / float64(total), nil
}
