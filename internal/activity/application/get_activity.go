package application

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

type GetActivityUseCase struct {
	activities domain.ActivityRepository
	cache      ActivityCache
}

func NewGetActivityUseCase(activities domain.ActivityRepository, cache ActivityCache) *GetActivityUseCase {
	return &GetActivityUseCase{activities: activities, cache: cache}
}

func (uc *GetActivityUseCase) Execute(ctx context.Context, id uuid.UUID) (*domain.Activity, error) {
	if activity, found, err := uc.cache.GetDetail(ctx, id); err != nil {
		log.Printf("activity: cache read failed for %s: %v", id, err)
	} else if found {
		return activity, nil
	}

	activity, err := uc.activities.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := uc.cache.SetDetail(ctx, activity); err != nil {
		log.Printf("activity: cache write failed for %s: %v", id, err)
	}

	return activity, nil
}
