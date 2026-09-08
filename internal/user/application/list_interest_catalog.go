package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/user/domain"
)

type ListInterestCatalogUseCase struct {
	interests domain.InterestRepository
}

func NewListInterestCatalogUseCase(interests domain.InterestRepository) *ListInterestCatalogUseCase {
	return &ListInterestCatalogUseCase{interests: interests}
}

func (uc *ListInterestCatalogUseCase) Execute(ctx context.Context) ([]domain.Interest, error) {
	return uc.interests.ListActive(ctx)
}
