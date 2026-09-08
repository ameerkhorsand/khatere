package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/auth/domain"

	"github.com/google/uuid"
)

type RegisterUseCase struct {
	accounts domain.AccountRepository
	hasher   domain.PasswordHasher
}

func NewRegisterUseCase(accounts domain.AccountRepository, hasher domain.PasswordHasher) *RegisterUseCase {
	return &RegisterUseCase{accounts: accounts, hasher: hasher}
}

type RegisterInput struct {
	Email       string
	Password    string
	AccountType domain.AccountType
}

func (uc *RegisterUseCase) Execute(ctx context.Context, in RegisterInput) (*domain.Account, error) {
	if !in.AccountType.Valid() {
		return nil, domain.ErrInvalidAccountType
	}

	existing, err := uc.accounts.FindByEmail(ctx, in.Email)
	if err != nil && err != domain.ErrAccountNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	account := &domain.Account{
		ID:           uuid.New(),
		Email:        in.Email,
		PasswordHash: hash,
		AccountType:  in.AccountType,
		Metadata:     map[string]any{},
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.accounts.Create(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}
