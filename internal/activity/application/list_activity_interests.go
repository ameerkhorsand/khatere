package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

// ListActivityInterestsUseCase is a simple passthrough — same visibility
// as reading the activity itself, so no auth restriction here.
type ListActivityInterestsUseCase struct {
	interests domain.ActivityInterestRepository
}

func NewListActivityInterestsUseCase(interests domain.ActivityInterestRepository) *ListActivityInterestsUseCase {
	return &ListActivityInterestsUseCase{interests: interests}
}

func (uc *ListActivityInterestsUseCase) Execute(ctx context.Context, activityID uuid.UUID) ([]domain.Interest, error) {
	return uc.interests.FindByActivityID(ctx, activityID)
}
