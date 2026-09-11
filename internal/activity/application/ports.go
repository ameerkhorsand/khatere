package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

// NotificationPublisher sends one notification event onward, to
// Kafka in production. Satisfied directly by
// internal/notification/adapters/kafka.Producer — no translation
// adapter needed, since notification.Event is a shared, thin
// cross-domain contract, not this domain's own concrete type.
type NotificationPublisher interface {
	Publish(ctx context.Context, event notificationdomain.Event) error
}

// ActivityCache is a fast, disposable read-through cache in front of
// ActivityRepository.FindByID and .List (approved-only). It is never
// the source of truth — Postgres is — so a miss or an entirely empty
// cache must always be a safe, correct fallback, never an error.
//
// One interface, not two, because ApproveActivityUseCase needs both
// halves at once (a newly-approved activity affects its own detail
// AND the approved list), while RejectActivityUseCase and
// CreateActivityUseCase each only need one half. A single dependency
// is simpler for the use cases that need both.
type ActivityCache interface {
	GetDetail(ctx context.Context, activityID uuid.UUID) (activity *domain.Activity, found bool, err error)
	SetDetail(ctx context.Context, activity *domain.Activity) error
	InvalidateDetail(ctx context.Context, activityID uuid.UUID) error

	// GetList/SetList cache the single "approved, non-deleted
	// activities" list — there's only one such list today, since
	// ListActivitiesUseCase takes no filter parameters.
	GetList(ctx context.Context) (activities []domain.Activity, found bool, err error)
	SetList(ctx context.Context, activities []domain.Activity) error
	InvalidateList(ctx context.Context) error
}
