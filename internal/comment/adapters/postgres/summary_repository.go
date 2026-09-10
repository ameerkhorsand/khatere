package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentSummaryRepository struct {
	pool *pgxpool.Pool
}

func NewCommentSummaryRepository(pool *pgxpool.Pool) *CommentSummaryRepository {
	return &CommentSummaryRepository{pool: pool}
}

const summaryColumns = `activity_id, summary, comment_count, generated_at, created_at, updated_at`

// Upsert relies on the PRIMARY KEY on activity_id (migration 0017):
// a second write for the same activity overwrites the row in
// place, keeping created_at from the first write.
func (r *CommentSummaryRepository) Upsert(ctx context.Context, summary *domain.CommentSummary) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO comment_summaries (activity_id, summary, comment_count, generated_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())
		ON CONFLICT (activity_id)
		DO UPDATE SET
			summary       = EXCLUDED.summary,
			comment_count = EXCLUDED.comment_count,
			generated_at  = EXCLUDED.generated_at,
			updated_at    = now()
	`, summary.ActivityID, summary.Summary, summary.CommentCount, summary.GeneratedAt)
	return err
}

func (r *CommentSummaryRepository) FindByActivityID(ctx context.Context, activityID uuid.UUID) (*domain.CommentSummary, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+summaryColumns+` FROM comment_summaries WHERE activity_id = $1`,
		activityID)
	return scanCommentSummary(row)
}

func scanCommentSummary(row pgx.Row) (*domain.CommentSummary, error) {
	var s domain.CommentSummary
	err := row.Scan(&s.ActivityID, &s.Summary, &s.CommentCount, &s.GeneratedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSummaryNotFound
		}
		return nil, err
	}
	return &s, nil
}
