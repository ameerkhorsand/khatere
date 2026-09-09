package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

type ApproveActivityUseCase struct {
	activities domain.ActivityRepository
}

func NewApproveActivityUseCase(activities domain.ActivityRepository) *ApproveActivityUseCase {
	return &ApproveActivityUseCase{activities: activities}
}

type ApproveActivityInput struct {
	ActivityID  uuid.UUID
	ModeratorID uuid.UUID
}

func (uc *ApproveActivityUseCase) Execute(ctx context.Context, in ApproveActivityInput) (*domain.Activity, error) {
	activity, err := uc.activities.FindByID(ctx, in.ActivityID)
	if err != nil {
		return nil, err
	}

	if activity.Status != domain.ActivityStatusPending {
		return nil, domain.ErrActivityNotPending
	}

	now := time.Now()
	activity.Status = domain.ActivityStatusApproved
	activity.ReviewedBy = &in.ModeratorID
	activity.ReviewedAt = &now

	if err := uc.activities.Update(ctx, activity); err != nil {
		return nil, err
	}

	return activity, nil
}
