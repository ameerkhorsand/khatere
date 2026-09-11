package application

import (
	"context"

	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
)

// NotificationPublisher sends one notification event onward, to
// Kafka in production. Satisfied directly by
// internal/notification/adapters/kafka.Producer — no translation
// adapter needed, since notification.Event is a shared, thin
// cross-domain contract, not this domain's own concrete type.
type NotificationPublisher interface {
	Publish(ctx context.Context, event notificationdomain.Event) error
}
