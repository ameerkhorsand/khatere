package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RatingRepository struct {
	pool *pgxpool.Pool
}

func NewRatingRepository(pool *pgxpool.Pool) *RatingRepository {
	return &RatingRepository{pool: pool}
}

const ratingColumns = `id, activity_id, user_id, score, created_at, updated_at`

// Upsert relies on the UNIQUE(activity_id, user_id) constraint from
// migration 0014: a second write for the same pair overwrites the
// score and updated_at in place, keeping the original id and
// created_at.
func (r *RatingRepository) Upsert(ctx context.Context, rating *domain.Rating) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ratings (id, activity_id, user_id, score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (activity_id, user_id)
		DO UPDATE SET score = EXCLUDED.score, updated_at = EXCLUDED.updated_at
	`, rating.ID, rating.ActivityID, rating.UserID, rating.Score, rating.CreatedAt, rating.UpdatedAt)
	return err
}

func (r *RatingRepository) FindByActivityAndUser(ctx context.Context, activityID, userID uuid.UUID) (*domain.Rating, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+ratingColumns+` FROM ratings WHERE activity_id = $1 AND user_id = $2`,
		activityID, userID)
	return scanRating(row)
}

func (r *RatingRepository) Summarize(ctx context.Context, activityID uuid.UUID) (domain.Summary, error) {
	var summary domain.Summary
	// COALESCE covers the zero-ratings case: AVG(score) is NULL
	// with no rows, and COUNT(*) is already 0 on its own.
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(score), 0), COUNT(*)
		FROM ratings
		WHERE activity_id = $1
	`, activityID).Scan(&summary.Average, &summary.Count)
	return summary, err
}

func scanRating(row pgx.Row) (*domain.Rating, error) {
	var rt domain.Rating
	err := row.Scan(&rt.ID, &rt.ActivityID, &rt.UserID, &rt.Score, &rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRatingNotFound
		}
		return nil, err
	}
	return &rt, nil
}
