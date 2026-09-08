package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/auth/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, t *domain.RefreshToken) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, account_id, token_hash, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, t.ID, t.AccountID, t.TokenHash, t.ExpiresAt, t.Revoked, t.CreatedAt)
	return err
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, account_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens WHERE token_hash = $1
	`, tokenHash)

	var t domain.RefreshToken
	err := row.Scan(&t.ID, &t.AccountID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked = true WHERE id = $1`, id)
	return err
}
