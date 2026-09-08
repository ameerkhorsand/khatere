package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/auth/domain"

	"github.com/google/uuid"
)

type RefreshUseCase struct {
	accounts      domain.AccountRepository
	refreshTokens domain.RefreshTokenRepository
	tokens        domain.TokenService
	refreshTTL    time.Duration
}

func NewRefreshUseCase(
	accounts domain.AccountRepository,
	refreshTokens domain.RefreshTokenRepository,
	tokens domain.TokenService,
	refreshTTL time.Duration,
) *RefreshUseCase {
	return &RefreshUseCase{accounts, refreshTokens, tokens, refreshTTL}
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
}

// Execute rotates the refresh token: the old one is revoked, a new one issued.
func (uc *RefreshUseCase) Execute(ctx context.Context, refreshPlain string) (*RefreshOutput, error) {
	hash := uc.tokens.HashRefreshToken(refreshPlain)

	stored, err := uc.refreshTokens.FindByHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrRefreshTokenInvalid
	}
	if stored.Revoked {
		return nil, domain.ErrRefreshTokenRevoked
	}
	if stored.Expired() {
		return nil, domain.ErrRefreshTokenInvalid
	}

	account, err := uc.accounts.FindByID(ctx, stored.AccountID)
	if err != nil || account.IsDeleted() {
		return nil, domain.ErrRefreshTokenInvalid
	}

	if err := uc.refreshTokens.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}

	accessToken, err := uc.tokens.IssueAccessToken(account.ID, account.AccountType)
	if err != nil {
		return nil, err
	}

	newRefreshPlain, err := uc.tokens.GenerateRefreshTokenValue()
	if err != nil {
		return nil, err
	}

	newRT := &domain.RefreshToken{
		ID:        uuid.New(),
		AccountID: account.ID,
		TokenHash: uc.tokens.HashRefreshToken(newRefreshPlain),
		ExpiresAt: time.Now().Add(uc.refreshTTL),
		CreatedAt: time.Now(),
	}
	if err := uc.refreshTokens.Create(ctx, newRT); err != nil {
		return nil, err
	}

	return &RefreshOutput{AccessToken: accessToken, RefreshToken: newRefreshPlain}, nil
}
