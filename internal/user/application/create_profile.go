package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
)

type CreateProfileUseCase struct {
	users domain.UserRepository
}

func NewCreateProfileUseCase(users domain.UserRepository) *CreateProfileUseCase {
	return &CreateProfileUseCase{users: users}
}

type CreateProfileInput struct {
	AccountID   uuid.UUID
	DisplayName string
	Bio         *string
}

func (uc *CreateProfileUseCase) Execute(ctx context.Context, in CreateProfileInput) (*domain.User, error) {
	existing, err := uc.users.FindByAccountID(ctx, in.AccountID)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	user := &domain.User{
		AccountID:   in.AccountID,
		DisplayName: in.DisplayName,
		Bio:         in.Bio,
		Metadata:    map[string]any{},
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
