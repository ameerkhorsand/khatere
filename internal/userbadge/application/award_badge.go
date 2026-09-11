package application

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
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
	notify     NotificationPublisher
}

func NewAwardBadgeUseCase(userBadges domain.UserBadgeRepository, badges BadgeLookup, notify NotificationPublisher) *AwardBadgeUseCase {
	return &AwardBadgeUseCase{userBadges: userBadges, badges: badges, notify: notify}
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
	if err := uc.userBadges.Create(ctx, award); err != nil {
		return err
	}

	metadata, _ := json.Marshal(map[string]string{
		"badge_id": badgeID.String(),
		"name":     name,
		"icon_key": iconKey,
	})
	if err := uc.notify.Publish(ctx, notificationdomain.Event{
		Type: notificationdomain.TypeBadgeEarned,
		// No human actor caused this — awarding is a system action
		// triggered by a QR scan, not a person doing something to
		// another person. Using the recipient as their own actor
		// avoids a nullable actor column for this one case.
		RecipientID: userID,
		ActorID:     userID,
		Metadata:    metadata,
	}); err != nil {
		log.Printf("userbadge: failed to publish badge-earned notification for %s: %v", award.ID, err)
	}

	return nil
}
