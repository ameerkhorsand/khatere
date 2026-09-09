package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/google/uuid"
)

type GetRatingSummaryUseCase struct {
	ratings domain.RatingRepository
}

func NewGetRatingSummaryUseCase(ratings domain.RatingRepository) *GetRatingSummaryUseCase {
	return &GetRatingSummaryUseCase{ratings: ratings}
}

func (uc *GetRatingSummaryUseCase) Execute(ctx context.Context, activityID uuid.UUID) (domain.Summary, error) {
	return uc.ratings.Summarize(ctx, activityID)
}
