package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bLorax/khatere-backend/internal/auth/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Create(ctx context.Context, a *domain.Account) error {
	meta, err := json.Marshal(a.Metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO accounts (id, email, password_hash, account_type, metadata, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, a.ID, a.Email, a.PasswordHash, a.AccountType, meta, a.Version, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *AccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	return r.scanOne(ctx, `
		SELECT id, email, password_hash, account_type, metadata, version, created_at, updated_at, deleted_at
		FROM accounts WHERE id = $1
	`, id)
}

func (r *AccountRepository) FindByEmail(ctx context.Context, email string) (*domain.Account, error) {
	return r.scanOne(ctx, `
		SELECT id, email, password_hash, account_type, metadata, version, created_at, updated_at, deleted_at
		FROM accounts WHERE email = $1
	`, email)
}

func (r *AccountRepository) scanOne(ctx context.Context, query string, arg any) (*domain.Account, error) {
	row := r.pool.QueryRow(ctx, query, arg)

	var a domain.Account
	var metaBytes []byte
	err := row.Scan(&a.ID, &a.Email, &a.PasswordHash, &a.AccountType, &metaBytes,
		&a.Version, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &a.Metadata); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AccountRepository) Update(ctx context.Context, a *domain.Account) error {
	meta, err := json.Marshal(a.Metadata)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounts
		SET password_hash = $1, metadata = $2, deleted_at = $3
		WHERE id = $4 AND version = $5
	`, a.PasswordHash, meta, a.DeletedAt, a.ID, a.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}
