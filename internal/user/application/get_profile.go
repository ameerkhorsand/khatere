package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
)

type GetProfileUseCase struct {
	users domain.UserRepository
}

func NewGetProfileUseCase(users domain.UserRepository) *GetProfileUseCase {
	return &GetProfileUseCase{users: users}
}

// Execute returns the user profile for the given account ID.
// It returns domain.ErrUserNotFound when no profile exists yet.
func (uc *GetProfileUseCase) Execute(ctx context.Context, accountID uuid.UUID) (*domain.User, error) {
	user, err := uc.users.FindByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
