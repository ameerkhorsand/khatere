package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/moderator/domain"
	"github.com/google/uuid"
)

type CreateModeratorProfileUseCase struct {
	moderators domain.ModeratorRepository
}

func NewCreateModeratorProfileUseCase(moderators domain.ModeratorRepository) *CreateModeratorProfileUseCase {
	return &CreateModeratorProfileUseCase{moderators: moderators}
}

type CreateModeratorProfileInput struct {
	AccountID uuid.UUID
}

func (uc *CreateModeratorProfileUseCase) Execute(ctx context.Context, in CreateModeratorProfileInput) (*domain.Moderator, error) {
	existing, err := uc.moderators.FindByAccountID(ctx, in.AccountID)
	if err != nil && err != domain.ErrModeratorNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrModeratorAlreadyExists
	}

	moderator := &domain.Moderator{
		AccountID: in.AccountID,
		Metadata:  map[string]any{},
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.moderators.Create(ctx, moderator); err != nil {
		return nil, err
	}

	return moderator, nil
}
