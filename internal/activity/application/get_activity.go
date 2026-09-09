package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

type GetActivityUseCase struct {
	activities domain.ActivityRepository
}

func NewGetActivityUseCase(activities domain.ActivityRepository) *GetActivityUseCase {
	return &GetActivityUseCase{activities: activities}
}

func (uc *GetActivityUseCase) Execute(ctx context.Context, id uuid.UUID) (*domain.Activity, error) {
	return uc.activities.FindByID(ctx, id)
}
