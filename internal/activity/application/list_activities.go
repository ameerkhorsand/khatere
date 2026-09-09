package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
)

type ListActivitiesUseCase struct {
	activities domain.ActivityRepository
}

func NewListActivitiesUseCase(activities domain.ActivityRepository) *ListActivitiesUseCase {
	return &ListActivitiesUseCase{activities: activities}
}

// Execute lists only approved activities — this is the public/Discovery
// listing. Use ListModerationQueueUseCase for the moderator's pending view.
func (uc *ListActivitiesUseCase) Execute(ctx context.Context) ([]domain.Activity, error) {
	approved := domain.ActivityStatusApproved
	return uc.activities.List(ctx, domain.ListFilter{Status: &approved})
}
