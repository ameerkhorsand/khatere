package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

type RejectActivityUseCase struct {
	activities domain.ActivityRepository
	notify     NotificationPublisher
	cache      ActivityCache
}

func NewRejectActivityUseCase(activities domain.ActivityRepository, notify NotificationPublisher, cache ActivityCache) *RejectActivityUseCase {
	return &RejectActivityUseCase{activities: activities, notify: notify, cache: cache}
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

	// A rejected activity was never in the approved list, so only
	// its own detail entry (if it was ever cached, e.g. previewed by
	// a moderator) needs invalidating — no list invalidation here.
	if err := uc.cache.InvalidateDetail(ctx, activity.ID); err != nil {
		log.Printf("activity: detail cache invalidation failed for %s: %v", activity.ID, err)
	}

	metadata, _ := json.Marshal(map[string]string{
		"activity_id":    activity.ID.String(),
		"activity_title": activity.Title,
		"reason":         reasonOrEmpty(activity.RejectionReason),
	})
	if err := uc.notify.Publish(ctx, notificationdomain.Event{
		Type:        notificationdomain.TypeActivityRejected,
		RecipientID: activity.CreatedBy,
		ActorID:     in.ModeratorID,
		Metadata:    metadata,
	}); err != nil {
		log.Printf("activity: failed to publish rejection notification for %s: %v", activity.ID, err)
	}

	return activity, nil
}

// reasonOrEmpty reads a rejection reason safely. Reason is a
// *string, so json.Marshal on a nil pointer would put null in the
// metadata map instead of an empty string.
func reasonOrEmpty(reason *string) string {
	if reason == nil {
		return ""
	}
	return *reason
}
