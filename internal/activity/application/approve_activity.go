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

type ApproveActivityUseCase struct {
	activities domain.ActivityRepository
	notify     NotificationPublisher
	cache      ActivityCache
}

func NewApproveActivityUseCase(activities domain.ActivityRepository, notify NotificationPublisher, cache ActivityCache) *ApproveActivityUseCase {
	return &ApproveActivityUseCase{activities: activities, notify: notify, cache: cache}
}

type ApproveActivityInput struct {
	ActivityID  uuid.UUID
	ModeratorID uuid.UUID
}

func (uc *ApproveActivityUseCase) Execute(ctx context.Context, in ApproveActivityInput) (*domain.Activity, error) {
	activity, err := uc.activities.FindByID(ctx, in.ActivityID)
	if err != nil {
		return nil, err
	}

	if activity.Status != domain.ActivityStatusPending {
		return nil, domain.ErrActivityNotPending
	}

	now := time.Now()
	activity.Status = domain.ActivityStatusApproved
	activity.ReviewedBy = &in.ModeratorID
	activity.ReviewedAt = &now

	if err := uc.activities.Update(ctx, activity); err != nil {
		return nil, err
	}

	// Best-effort, same reasoning as the notification publish below:
	// the approval already succeeded, so a cache invalidation failure
	// must not undo that. Both entries need invalidating — the
	// activity's own detail, and the approved list it just joined.
	if err := uc.cache.InvalidateDetail(ctx, activity.ID); err != nil {
		log.Printf("activity: detail cache invalidation failed for %s: %v", activity.ID, err)
	}
	if err := uc.cache.InvalidateList(ctx); err != nil {
		log.Printf("activity: list cache invalidation failed: %v", err)
	}

	// Best-effort, same as ApproveCommentUseCase's summary
	// regeneration: the approval already succeeded, so a
	// notification failure must never make it look like it didn't.
	metadata, _ := json.Marshal(map[string]string{
		"activity_id":    activity.ID.String(),
		"activity_title": activity.Title,
	})
	if err := uc.notify.Publish(ctx, notificationdomain.Event{
		Type:        notificationdomain.TypeActivityApproved,
		RecipientID: activity.CreatedBy,
		ActorID:     in.ModeratorID,
		Metadata:    metadata,
	}); err != nil {
		log.Printf("activity: failed to publish approval notification for %s: %v", activity.ID, err)
	}

	return activity, nil
}
