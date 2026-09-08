package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
)

type UpdateProfileUseCase struct {
	users domain.UserRepository
}

func NewUpdateProfileUseCase(users domain.UserRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{users: users}
}

type UpdateProfileInput struct {
	AccountID         uuid.UUID
	DisplayName       *string // nil = leave unchanged
	Bio               *string
	ProfilePictureKey *string
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, in UpdateProfileInput) (*domain.User, error) {
	user, err := uc.users.FindByAccountID(ctx, in.AccountID)
	if err != nil {
		return nil, err
	}

	if in.DisplayName != nil {
		user.DisplayName = *in.DisplayName
	}
	if in.Bio != nil {
		user.Bio = in.Bio
	}
	if in.ProfilePictureKey != nil {
		user.ProfilePictureKey = in.ProfilePictureKey
	}

	if err := uc.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
