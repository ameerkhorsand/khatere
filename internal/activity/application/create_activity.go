package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	"github.com/google/uuid"
)

type CreateActivityUseCase struct {
	activities domain.ActivityRepository
}

func NewCreateActivityUseCase(activities domain.ActivityRepository) *CreateActivityUseCase {
	return &CreateActivityUseCase{activities: activities}
}

type CreateActivityInput struct {
	Title       string
	Description *string
	SourceType  domain.SourceType
	CreatedBy   uuid.UUID
}

func (uc *CreateActivityUseCase) Execute(ctx context.Context, in CreateActivityInput) (*domain.Activity, error) {
	if !in.SourceType.Valid() {
		return nil, domain.ErrInvalidSourceType
	}

	// Moderator- and host-created activities are shown immediately.
	// User-generated activities require moderator approval first.
	status := domain.ActivityStatusApproved
	if in.SourceType == domain.SourceTypeUser {
		status = domain.ActivityStatusPending
	}

	activity := &domain.Activity{
		ID:          uuid.New(),
		Title:       in.Title,
		Description: in.Description,
		SourceType:  in.SourceType,
		CreatedBy:   in.CreatedBy,
		Status:      status,
		Metadata:    map[string]any{},
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.activities.Create(ctx, activity); err != nil {
		return nil, err
	}

	return activity, nil
}
