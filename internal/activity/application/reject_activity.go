package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

type RejectActivityUseCase struct {
	activities domain.ActivityRepository
}

func NewRejectActivityUseCase(activities domain.ActivityRepository) *RejectActivityUseCase {
	return &RejectActivityUseCase{activities: activities}
}

type RejectActivityInput struct {
	ActivityID  uuid.UUID
	ModeratorID uuid.UUID
	Reason      *string
}

func (uc *RejectActivityUseCase) Execute(ctx context.Context, in RejectActivityInput) (*domain.Activity, error) {
	activity, err := uc.activities.FindByID(ctx, in.ActivityID)
	if err != nil {
		return nil, err
	}

	if activity.Status != domain.ActivityStatusPending {
		return nil, domain.ErrActivityNotPending
	}

	now := time.Now()
	activity.Status = domain.ActivityStatusRejected
	activity.ReviewedBy = &in.ModeratorID
	activity.ReviewedAt = &now
	activity.RejectionReason = in.Reason

	if err := uc.activities.Update(ctx, activity); err != nil {
		return nil, err
	}

	return activity, nil
}
