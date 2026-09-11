package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

type ListMyActivitiesUseCase struct {
	activities domain.ActivityRepository
}

func NewListMyActivitiesUseCase(activities domain.ActivityRepository) *ListMyActivitiesUseCase {
	return &ListMyActivitiesUseCase{activities: activities}
}

// Execute lists activities created by the given account, regardless of
// status — unlike ListActivitiesUseCase, which only returns approved
// activities for the public/Discovery listing. This lets a creator see
// their own pending and rejected activities too.
func (uc *ListMyActivitiesUseCase) Execute(ctx context.Context, accountID uuid.UUID) ([]domain.Activity, error) {
	return uc.activities.List(ctx, domain.ListFilter{CreatedBy: &accountID})
}
