package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/notification/domain"
)

// EventPublisher sends one notification-worthy event onward — to
// Kafka in production, to nothing in a test double. Every domain
// that triggers a notification (hangout, archive, activity, comment,
// badge) gets its own narrow adapter that calls this, rather than
// importing this package's Kafka adapter directly — see each
// domain's own notifier port, added in Step 9.
type EventPublisher interface {
	Publish(ctx context.Context, event domain.Event) error
}
