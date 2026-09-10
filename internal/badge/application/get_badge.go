package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/badge/domain"
	"github.com/google/uuid"
)

type GetBadgeUseCase struct {
	badges     domain.BadgeRepository
	activities ActivityOwnershipChecker
}

func NewGetBadgeUseCase(badges domain.BadgeRepository, activities ActivityOwnershipChecker) *GetBadgeUseCase {
	return &GetBadgeUseCase{badges: badges, activities: activities}
}

type GetBadgeInput struct {
	ActivityID uuid.UUID
	HostID     uuid.UUID
}

// Execute returns the host's own badge for the Host Dashboard setup
// screen. Only the owning host may fetch it this way — a user who
// earned the badge reads it through their own UserBadge record
// instead (internal/userbadge), which already carries a name/icon
// snapshot and needs no call into this domain.
func (uc *GetBadgeUseCase) Execute(ctx context.Context, in GetBadgeInput) (*domain.Badge, error) {
	owned, err := uc.activities.IsOwnedByHost(ctx, in.ActivityID, in.HostID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, domain.ErrNotHostActivity
	}
	return uc.badges.FindByActivityID(ctx, in.ActivityID)
}
