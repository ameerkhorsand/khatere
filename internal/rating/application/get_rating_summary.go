package application

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/google/uuid"
)

type GetRatingSummaryUseCase struct {
	ratings domain.RatingRepository
	cache   SummaryCache
}

func NewGetRatingSummaryUseCase(ratings domain.RatingRepository, cache SummaryCache) *GetRatingSummaryUseCase {
	return &GetRatingSummaryUseCase{ratings: ratings, cache: cache}
}

func (uc *GetRatingSummaryUseCase) Execute(ctx context.Context, activityID uuid.UUID) (domain.Summary, error) {
	if summary, found, err := uc.cache.Get(ctx, activityID); err != nil {
		// A cache error must not fail the request — fall through to
		// Postgres, same "fail open" reasoning as the rate limiter.
		log.Printf("rating: cache read failed for %s: %v", activityID, err)
	} else if found {
		return summary, nil
	}

	summary, err := uc.ratings.Summarize(ctx, activityID)
	if err != nil {
		return domain.Summary{}, err
	}

	if err := uc.cache.Set(ctx, activityID, summary); err != nil {
		log.Printf("rating: cache write failed for %s: %v", activityID, err)
	}

	return summary, nil
}
