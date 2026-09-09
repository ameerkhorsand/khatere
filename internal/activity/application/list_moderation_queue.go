package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
)

type ListModerationQueueUseCase struct {
	activities domain.ActivityRepository
}

func NewListModerationQueueUseCase(activities domain.ActivityRepository) *ListModerationQueueUseCase {
	return &ListModerationQueueUseCase{activities: activities}
}

func (uc *ListModerationQueueUseCase) Execute(ctx context.Context) ([]domain.Activity, error) {
	return uc.activities.ListPendingQueue(ctx)
}
