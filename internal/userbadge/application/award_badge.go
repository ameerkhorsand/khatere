package application

import (
	"context"
	"errors"
	"time"

	"github.com/bLorax/khatere-backend/internal/userbadge/domain"
	"github.com/google/uuid"
)

// BadgeLookup is intentionally small so this use case doesn't
// depend on the concrete badge repository, and never needs to
// import badge/domain — same reasoning as attendance's
// QRCodeLookup. A purpose-built method on a Postgres adapter in
// this package satisfies it (see badge_lookup_repository.go).
type BadgeLookup interface {
	// FindByActivityID reports the badge set up for activityID.
	// found is false with a nil err when the activity simply has
	// no badge — that's normal, not a system error.
	FindByActivityID(ctx context.Context, activityID uuid.UUID) (badgeID uuid.UUID, name, iconKey string, found bool, err error)
}

// AwardBadgeUseCase satisfies attendance's BadgeAwarder interface
// (internal/attendance/application/verify_scan.go).
type AwardBadgeUseCase struct {
	userBadges domain.UserBadgeRepository
	badges     BadgeLookup
}

func NewAwardBadgeUseCase(userBadges domain.UserBadgeRepository, badges BadgeLookup) *AwardBadgeUseCase {
	return &AwardBadgeUseCase{userBadges: userBadges, badges: badges}
}

// AwardIfBadgeExists gives userID the badge set up for activityID,
// if the host has set one up. A no-op, not an error, when the
// activity has no badge, or when the user already has it.
func (uc *AwardBadgeUseCase) AwardIfBadgeExists(ctx context.Context, userID, activityID uuid.UUID) error {
	badgeID, name, iconKey, found, err := uc.badges.FindByActivityID(ctx, activityID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	existing, err := uc.userBadges.FindByUserAndBadge(ctx, userID, badgeID)
	if err != nil && !errors.Is(err, domain.ErrUserBadgeNotFound) {
		return err
	}
	if existing != nil {
		return nil
	}

	award := &domain.UserBadge{
		ID:              uuid.New(),
		UserID:          userID,
		BadgeID:         &badgeID,
		ActivityID:      &activityID,
		NameSnapshot:    name,
		IconKeySnapshot: iconKey,
		Visible:         true,
		AwardedAt:       time.Now(),
	}
	return uc.userBadges.Create(ctx, award)
}
