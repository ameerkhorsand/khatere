package postgres

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InterestRepository struct {
	pool *pgxpool.Pool
}

func NewInterestRepository(pool *pgxpool.Pool) *InterestRepository {
	return &InterestRepository{pool: pool}
}

func (r *InterestRepository) ListActive(ctx context.Context) ([]domain.Interest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, label, category, active, sort_order, created_at
		FROM interests WHERE active = true ORDER BY sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInterests(rows)
}

func (r *InterestRepository) ReplaceAll(ctx context.Context, userID uuid.UUID, interestIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, interestID := range interestIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (id, user_id, interest_id, created_at)
			VALUES ($1, $2, $3, now())
		`, uuid.New(), userID, interestID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *InterestRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Interest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.slug, i.label, i.category, i.active, i.sort_order, i.created_at
		FROM interests i
		JOIN user_interests ui ON ui.interest_id = i.id
		WHERE ui.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInterests(rows)
}

func scanInterests(rows pgx.Rows) ([]domain.Interest, error) {
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
