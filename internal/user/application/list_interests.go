package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
)

type ListInterestsUseCase struct {
	interests domain.InterestRepository
}

func NewListInterestsUseCase(interests domain.InterestRepository) *ListInterestsUseCase {
	return &ListInterestsUseCase{interests: interests}
}

func (uc *ListInterestsUseCase) Execute(ctx context.Context, userID uuid.UUID) ([]domain.Interest, error) {
	return uc.interests.FindByUserID(ctx, userID)
}
