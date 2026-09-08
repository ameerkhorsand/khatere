package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/auth/domain"

	"github.com/google/uuid"
)

type LoginUseCase struct {
	accounts      domain.AccountRepository
	refreshTokens domain.RefreshTokenRepository
	hasher        domain.PasswordHasher
	tokens        domain.TokenService
	refreshTTL    time.Duration
}

func NewLoginUseCase(
	accounts domain.AccountRepository,
	refreshTokens domain.RefreshTokenRepository,
	hasher domain.PasswordHasher,
	tokens domain.TokenService,
	refreshTTL time.Duration,
) *LoginUseCase {
	return &LoginUseCase{accounts, refreshTokens, hasher, tokens, refreshTTL}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	Account      *domain.Account
}

func (uc *LoginUseCase) Execute(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	account, err := uc.accounts.FindByEmail(ctx, in.Email)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if account.IsDeleted() {
		return nil, domain.ErrInvalidCredentials
	}
	if !uc.hasher.Verify(account.PasswordHash, in.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	accessToken, err := uc.tokens.IssueAccessToken(account.ID, account.AccountType)
	if err != nil {
		return nil, err
	}

	refreshPlain, err := uc.tokens.GenerateRefreshTokenValue()
	if err != nil {
		return nil, err
	}

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		AccountID: account.ID,
		TokenHash: uc.tokens.HashRefreshToken(refreshPlain),
		ExpiresAt: time.Now().Add(uc.refreshTTL),
		CreatedAt: time.Now(),
	}
	if err := uc.refreshTokens.Create(ctx, rt); err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshPlain,
		Account:      account,
	}, nil
}
