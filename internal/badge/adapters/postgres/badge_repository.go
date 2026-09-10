package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/badge/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BadgeRepository struct {
	pool *pgxpool.Pool
}

func NewBadgeRepository(pool *pgxpool.Pool) *BadgeRepository {
	return &BadgeRepository{pool: pool}
}

const badgeColumns = `id, activity_id, created_by, name, icon_key, version, created_at, updated_at`

func (r *BadgeRepository) Create(ctx context.Context, b *domain.Badge) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO badges (id, activity_id, created_by, name, icon_key, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, b.ID, b.ActivityID, b.CreatedBy, b.Name, b.IconKey, b.Version, b.CreatedAt, b.UpdatedAt)
	return err
}

func (r *BadgeRepository) FindByActivityID(ctx context.Context, activityID uuid.UUID) (*domain.Badge, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+badgeColumns+` FROM badges WHERE activity_id = $1`, activityID)
	return scanBadge(row)
}

func (r *BadgeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Badge, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+badgeColumns+` FROM badges WHERE id = $1`, id)
	return scanBadge(row)
}

func scanBadge(row pgx.Row) (*domain.Badge, error) {
	var b domain.Badge
	err := row.Scan(&b.ID, &b.ActivityID, &b.CreatedBy, &b.Name, &b.IconKey, &b.Version, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBadgeNotFound
		}
		return nil, err
	}
	return &b, nil
}
