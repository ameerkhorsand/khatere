package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// candidateLimit bounds how many activities get scored per run.
// GenerateSuggestionsUseCase does 3 signal queries per candidate
// (interest, history, quality — engagement is once per user), so an
// unbounded candidate set doesn't scale. 500 is a starting cap, not
// a measured one; a later pass can pre-filter by category or
// location before this limit starts to bite.
const candidateLimit = 500

// ActivityCandidateSource implements application.ActivityCandidateSource.
// The filter is deliberately minimal: approved and not deleted. It
// does not exclude activities the user has done before — recurring
// suggestions (e.g. hiking resurfacing for someone who keeps
// accepting it) are part of the product design, not a bug to filter
// out here. userID is accepted for interface stability even though
// today's filter doesn't use it; a later pass (blocked hosts, one
// candidate set per region) will.
type ActivityCandidateSource struct {
	pool *pgxpool.Pool
}

func NewActivityCandidateSource(pool *pgxpool.Pool) *ActivityCandidateSource {
	return &ActivityCandidateSource{pool: pool}
}

func (s *ActivityCandidateSource) CandidateActivities(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM activities
		WHERE status = 'approved' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1
	`, candidateLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
