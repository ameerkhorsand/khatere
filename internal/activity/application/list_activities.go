package application

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
)

type ListActivitiesUseCase struct {
	activities domain.ActivityRepository
	cache      ActivityCache
}

func NewListActivitiesUseCase(activities domain.ActivityRepository, cache ActivityCache) *ListActivitiesUseCase {
	return &ListActivitiesUseCase{activities: activities, cache: cache}
}

// Execute lists only approved activities — this is the public/Discovery
// listing. Use ListModerationQueueUseCase for the moderator's pending view.
func (uc *ListActivitiesUseCase) Execute(ctx context.Context) ([]domain.Activity, error) {
	if activities, found, err := uc.cache.GetList(ctx); err != nil {
		log.Printf("activity: list cache read failed: %v", err)
	} else if found {
		return activities, nil
	}

	approved := domain.ActivityStatusApproved
	activities, err := uc.activities.List(ctx, domain.ListFilter{Status: &approved})
	if err != nil {
		return nil, err
	}

	if err := uc.cache.SetList(ctx, activities); err != nil {
		log.Printf("activity: list cache write failed: %v", err)
	}

	return activities, nil
}
