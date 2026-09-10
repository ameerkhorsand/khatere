package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/userbadge/domain"
	"github.com/google/uuid"
)

type ListUserBadgesUseCase struct {
	userBadges domain.UserBadgeRepository
}

func NewListUserBadgesUseCase(userBadges domain.UserBadgeRepository) *ListUserBadgesUseCase {
	return &ListUserBadgesUseCase{userBadges: userBadges}
}

// Execute returns everything userID has earned, for their own
// Profile "Enjoyed" section. This always returns the full set,
// hidden badges included, so the user's own management screen can
// show what they've chosen to hide. The frontend filters by
// Visible if it only wants what's shown to other people.
func (uc *ListUserBadgesUseCase) Execute(ctx context.Context, userID uuid.UUID) ([]domain.UserBadge, error) {
	return uc.userBadges.ListByUser(ctx, userID)
}
