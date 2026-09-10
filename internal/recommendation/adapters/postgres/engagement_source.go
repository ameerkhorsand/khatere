package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// engagementCircleThreshold and engagementHangoutThreshold are the
// counts at which each component saturates to 1.0. These are
// starting estimates, not measured — Step 8's rollout should revisit
// them once real usage data exists to calibrate against.
const (
	engagementCircleThreshold  = 20
	engagementHangoutThreshold = 5
)

// EngagementSource implements application.EngagementSource by
// combining circle size with recent hangout activity, independent of
// any single activity.
type EngagementSource struct {
	pool *pgxpool.Pool
}

func NewEngagementSource(pool *pgxpool.Pool) *EngagementSource {
	return &EngagementSource{pool: pool}
}

// Engagement averages two saturating components: circle size (how
// many accepted connections the user has) and recent hangout count
// (organized or joined as an accepted participant, in the last 30
// days). Each component is capped at 1.0 once it passes its
// threshold, so one very social user can't blow the scale out for
// everyone else.
func (s *EngagementSource) Engagement(ctx context.Context, userID uuid.UUID) (float64, error) {
	var circleSize, recentHangouts int
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM circle_connections
			 WHERE status = 'accepted'
			   AND (requester_id = $1 OR addressee_id = $1)) AS circle_size,
			(SELECT COUNT(*) FROM hangouts h
			 WHERE h.deleted_at IS NULL
			   AND h.created_at >= now() - INTERVAL '30 days'
			   AND (
			       h.organizer_id = $1
			       OR EXISTS (
			           SELECT 1 FROM hangout_participants hp
			           WHERE hp.hangout_id = h.id
			             AND hp.user_id = $1
			             AND hp.invite_status = 'accepted'
			       )
			   )) AS recent_hangouts
	`, userID).Scan(&circleSize, &recentHangouts)
	if err != nil {
		return 0, err
	}

	circleComponent := saturate(circleSize, engagementCircleThreshold)
	hangoutComponent := saturate(recentHangouts, engagementHangoutThreshold)

	return (circleComponent + hangoutComponent) / 2, nil
}

// saturate maps count to [0, 1], reaching 1.0 once count meets
// threshold.
func saturate(count, threshold int) float64 {
	if count >= threshold {
		return 1.0
	}
	return float64(count) / float64(threshold)
}
