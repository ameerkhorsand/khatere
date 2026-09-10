package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bLorax/khatere-backend/internal/recommendation/domain"
)

type RecommendationRepository struct {
	pool *pgxpool.Pool
}

func NewRecommendationRepository(pool *pgxpool.Pool) *RecommendationRepository {
	return &RecommendationRepository{pool: pool}
}

// ReplaceForUser deletes a user's existing cache rows and inserts the
// fresh set in one transaction, so a reader never sees a half-written
// cache (all-old or all-new, never a mix).
func (r *RecommendationRepository) ReplaceForUser(ctx context.Context, userID uuid.UUID, scores []domain.Score) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_recommendation_cache WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, score := range scores {
		signals, err := json.Marshal(score.Signals)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO user_recommendation_cache
				(user_id, activity_id, signals, total_score, generated_at)
			VALUES ($1, $2, $3, $4, $5)
		`, score.UserID, score.ActivityID, signals, score.Total, score.GeneratedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *RecommendationRepository) TopForUser(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Score, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, activity_id, signals, total_score, generated_at
		FROM user_recommendation_cache
		WHERE user_id = $1
		ORDER BY total_score DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanScores(rows)
}

func scanScores(rows pgx.Rows) ([]domain.Score, error) {
	scores := make([]domain.Score, 0)
	for rows.Next() {
		var s domain.Score
		var signalsBytes []byte
		if err := rows.Scan(&s.UserID, &s.ActivityID, &signalsBytes, &s.Total, &s.GeneratedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(signalsBytes, &s.Signals); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return scores, nil
}

// StaleUserIDs groups the cache by user and compares each user's
// newest row (MAX(generated_at)) against cutoff. Using the newest
// row, not the oldest, means a user only counts as stale once their
// whole cache set is out of date, not just one row within it.
func (r *RecommendationRepository) StaleUserIDs(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id
		FROM user_recommendation_cache
		GROUP BY user_id
		HAVING MAX(generated_at) < $1
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, rows.Err()
}
