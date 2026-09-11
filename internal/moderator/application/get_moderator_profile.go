package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/moderator/domain"
	"github.com/google/uuid"
)

type GetModeratorProfileUseCase struct {
	moderators domain.ModeratorRepository
}

func NewGetModeratorProfileUseCase(moderators domain.ModeratorRepository) *GetModeratorProfileUseCase {
	return &GetModeratorProfileUseCase{moderators: moderators}
}

// Execute returns the moderator profile for the given account ID.
// It returns domain.ErrModeratorNotFound when no profile exists yet.
func (uc *GetModeratorProfileUseCase) Execute(ctx context.Context, accountID uuid.UUID) (*domain.Moderator, error) {
	moderator, err := uc.moderators.FindByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return moderator, nil
}
