package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bLorax/khatere-backend/internal/moderator/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModeratorRepository struct {
	pool *pgxpool.Pool
}

func NewModeratorRepository(pool *pgxpool.Pool) *ModeratorRepository {
	return &ModeratorRepository{pool: pool}
}

func (r *ModeratorRepository) Create(ctx context.Context, m *domain.Moderator) error {
	meta, err := json.Marshal(m.Metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO moderators (account_id, metadata, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, m.AccountID, meta, m.Version, m.CreatedAt, m.UpdatedAt)
	return err
}

func (r *ModeratorRepository) FindByAccountID(ctx context.Context, accountID uuid.UUID) (*domain.Moderator, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT account_id, metadata, version, created_at, updated_at, deleted_at
		FROM moderators WHERE account_id = $1
	`, accountID)

	var m domain.Moderator
	var metaBytes []byte
	err := row.Scan(&m.AccountID, &metaBytes, &m.Version, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrModeratorNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &m.Metadata); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModeratorRepository) Update(ctx context.Context, m *domain.Moderator) error {
	meta, err := json.Marshal(m.Metadata)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE moderators SET metadata = $1, deleted_at = $2
		WHERE account_id = $3 AND version = $4
	`, meta, m.DeletedAt, m.AccountID, m.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}
