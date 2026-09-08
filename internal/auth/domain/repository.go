package domain

import (
	"context"

	"github.com/google/uuid"
)

type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	FindByID(ctx context.Context, id uuid.UUID) (*Account, error)
	FindByEmail(ctx context.Context, email string) (*Account, error)
	// Update performs an optimistic-lock update: it checks account.Version
	// against the stored row and returns ErrVersionConflict on mismatch.
	Update(ctx context.Context, account *Account) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type PasswordHasher interface {
	Hash(plaintext string) (string, error)
	Verify(hash, plaintext string) bool
}

type TokenService interface {
	IssueAccessToken(accountID uuid.UUID, accountType AccountType) (string, error)
	GenerateRefreshTokenValue() (string, error)
	HashRefreshToken(plaintext string) string
}
