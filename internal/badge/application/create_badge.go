package application

import (
	"context"
	"errors"
	"time"

	"github.com/bLorax/khatere-backend/internal/badge/domain"
	"github.com/google/uuid"
)

// ActivityOwnershipChecker is intentionally small so this use case
// doesn't depend on the concrete activity repository — same pattern
// as ActivityOwnershipChecker in the qrcode domain.
// ActivityRepository.IsOwnedByHost (internal/activity/adapters/postgres)
// satisfies this interface.
type ActivityOwnershipChecker interface {
	IsOwnedByHost(ctx context.Context, activityID, hostAccountID uuid.UUID) (bool, error)
}

type CreateBadgeUseCase struct {
	badges     domain.BadgeRepository
	activities ActivityOwnershipChecker
}

func NewCreateBadgeUseCase(badges domain.BadgeRepository, activities ActivityOwnershipChecker) *CreateBadgeUseCase {
	return &CreateBadgeUseCase{badges: badges, activities: activities}
}

type CreateBadgeInput struct {
	ActivityID uuid.UUID
	HostID     uuid.UUID
	Name       string
	IconKey    string
}

// Execute makes the one badge for a host's activity by hand. Unlike
// qrcode's Generate, a second call for the same activity is an
// error — the host's field values matter, so this never silently
// discards a new Name or IconKey.
func (uc *CreateBadgeUseCase) Execute(ctx context.Context, in CreateBadgeInput) (*domain.Badge, error) {
	owned, err := uc.activities.IsOwnedByHost(ctx, in.ActivityID, in.HostID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, domain.ErrNotHostActivity
	}

	existing, err := uc.badges.FindByActivityID(ctx, in.ActivityID)
	if err != nil && !errors.Is(err, domain.ErrBadgeNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrBadgeAlreadyExists
	}

	badge := &domain.Badge{
		ID:         uuid.New(),
		ActivityID: in.ActivityID,
		CreatedBy:  in.HostID,
		Name:       in.Name,
		IconKey:    in.IconKey,
		Version:    1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := uc.badges.Create(ctx, badge); err != nil {
		return nil, err
	}
	return badge, nil
}
