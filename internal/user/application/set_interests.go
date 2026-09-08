package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
)

type SetInterestsUseCase struct {
	interests domain.InterestRepository
}

func NewSetInterestsUseCase(interests domain.InterestRepository) *SetInterestsUseCase {
	return &SetInterestsUseCase{interests: interests}
}

type SetInterestsInput struct {
	UserID      uuid.UUID
	InterestIDs []uuid.UUID
}

func (uc *SetInterestsUseCase) Execute(ctx context.Context, in SetInterestsInput) error {
	return uc.interests.ReplaceAll(ctx, in.UserID, in.InterestIDs)
}
