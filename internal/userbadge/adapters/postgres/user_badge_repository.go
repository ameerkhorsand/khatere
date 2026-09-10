package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/userbadge/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserBadgeRepository struct {
	pool *pgxpool.Pool
}

func NewUserBadgeRepository(pool *pgxpool.Pool) *UserBadgeRepository {
	return &UserBadgeRepository{pool: pool}
}

const userBadgeColumns = `id, user_id, badge_id, activity_id, name_snapshot, icon_key_snapshot, visible, awarded_at`

func (r *UserBadgeRepository) Create(ctx context.Context, ub *domain.UserBadge) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_badges (id, user_id, badge_id, activity_id, name_snapshot, icon_key_snapshot, visible, awarded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, ub.ID, ub.UserID, ub.BadgeID, ub.ActivityID, ub.NameSnapshot, ub.IconKeySnapshot, ub.Visible, ub.AwardedAt)
	return err
}

func (r *UserBadgeRepository) FindByUserAndBadge(ctx context.Context, userID, badgeID uuid.UUID) (*domain.UserBadge, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userBadgeColumns+` FROM user_badges WHERE user_id = $1 AND badge_id = $2`,
		userID, badgeID)
	return scanUserBadge(row)
}

func (r *UserBadgeRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.UserBadge, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userBadgeColumns+` FROM user_badges WHERE user_id = $1 ORDER BY awarded_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.UserBadge
	for rows.Next() {
		ub, err := scanUserBadge(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *ub)
	}
	return result, rows.Err()
}

func scanUserBadge(row pgx.Row) (*domain.UserBadge, error) {
	var ub domain.UserBadge
	err := row.Scan(&ub.ID, &ub.UserID, &ub.BadgeID, &ub.ActivityID, &ub.NameSnapshot, &ub.IconKeySnapshot, &ub.Visible, &ub.AwardedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserBadgeNotFound
		}
		return nil, err
	}
	return &ub, nil
}
