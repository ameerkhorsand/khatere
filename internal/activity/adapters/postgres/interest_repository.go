package postgres

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivityInterestRepository implements domain.ActivityInterestRepository.
// Mirrors internal/user/adapters/postgres/interest_repository.go — same
// ReplaceAll/FindBy shape, different owner column (activity_id instead
// of user_id).
type ActivityInterestRepository struct {
	pool *pgxpool.Pool
}

func NewActivityInterestRepository(pool *pgxpool.Pool) *ActivityInterestRepository {
	return &ActivityInterestRepository{pool: pool}
}

func (r *ActivityInterestRepository) ReplaceAll(ctx context.Context, activityID uuid.UUID, interestIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM activity_interests WHERE activity_id = $1`, activityID); err != nil {
		return err
	}

	for _, interestID := range interestIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO activity_interests (id, activity_id, interest_id, created_at)
			VALUES ($1, $2, $3, now())
		`, uuid.New(), activityID, interestID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ActivityInterestRepository) FindByActivityID(ctx context.Context, activityID uuid.UUID) ([]domain.Interest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.slug, i.label, i.category, i.active, i.sort_order, i.created_at
		FROM interests i
		JOIN activity_interests ai ON ai.interest_id = i.id
		WHERE ai.activity_id = $1
	`, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivityInterests(rows)
}

func scanActivityInterests(rows pgx.Rows) ([]domain.Interest, error) {
	var out []domain.Interest
	for rows.Next() {
		var i domain.Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label, &i.Category, &i.Active, &i.SortOrder, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
