package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bLorax/khatere-backend/internal/host/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HostRepository struct {
	pool *pgxpool.Pool
}

func NewHostRepository(pool *pgxpool.Pool) *HostRepository {
	return &HostRepository{pool: pool}
}

func (r *HostRepository) Create(ctx context.Context, h *domain.Host) error {
	meta, err := json.Marshal(h.Metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO hosts (account_id, business_name, location_info, metadata, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, h.AccountID, h.BusinessName, h.LocationInfo, meta, h.Version, h.CreatedAt, h.UpdatedAt)
	return err
}

func (r *HostRepository) FindByAccountID(ctx context.Context, accountID uuid.UUID) (*domain.Host, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT account_id, business_name, location_info, metadata, version, created_at, updated_at, deleted_at
		FROM hosts WHERE account_id = $1
	`, accountID)

	var h domain.Host
	var metaBytes []byte
	err := row.Scan(&h.AccountID, &h.BusinessName, &h.LocationInfo, &metaBytes,
		&h.Version, &h.CreatedAt, &h.UpdatedAt, &h.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHostNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &h.Metadata); err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *HostRepository) Update(ctx context.Context, h *domain.Host) error {
	meta, err := json.Marshal(h.Metadata)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE hosts
		SET business_name = $1, location_info = $2, metadata = $3, deleted_at = $4
		WHERE account_id = $5 AND version = $6
	`, h.BusinessName, h.LocationInfo, meta, h.DeletedAt, h.AccountID, h.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}
