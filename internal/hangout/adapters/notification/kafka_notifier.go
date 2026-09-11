package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	notificationapp "github.com/bLorax/khatere-backend/internal/notification/application"
	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
)

// typeMap translates hangout's own NotificationType into the shared
// notification.domain.Type. The string values happen to match today,
// but an explicit map keeps hangout's domain from needing to import
// notification's domain just to stay in sync, and fails loudly (via
// the ok check in Notify) if a new hangout type is ever added here
// without a matching entry.
var typeMap = map[domain.NotificationType]notificationdomain.Type{
	domain.NotificationHangoutInvite:      notificationdomain.TypeHangoutInvite,
	domain.NotificationInviteResponded:    notificationdomain.TypeInviteResponded,
	domain.NotificationHangoutCancelled:   notificationdomain.TypeHangoutCancelled,
	domain.NotificationMeetupPinProposed:  notificationdomain.TypeMeetupPinProposed,
	domain.NotificationMeetupPinChanged:   notificationdomain.TypeMeetupPinChanged,
	domain.NotificationMeetupPinConfirmed: notificationdomain.TypeMeetupPinConfirmed,
}

// KafkaNotifier implements domain.Notifier by translating a hangout
// Notification into the shared notification.Event and publishing it
// through the Kafka pipeline — Step 9's port, satisfied directly by
// internal/notification/adapters/kafka.Producer, same as every other
// domain's notifier adapter. This is the production replacement for
// LogNotifier: every hangout use case that depends on domain.Notifier
// (invite, respond, cancel, pin propose/change/confirm) needs no code
// change — only the adapter passed into them in server.go changes.
type KafkaNotifier struct {
	publish notificationapp.EventPublisher
}

func NewKafkaNotifier(publish notificationapp.EventPublisher) *KafkaNotifier {
	return &KafkaNotifier{publish: publish}
}

func (n *KafkaNotifier) Notify(ctx context.Context, ntf domain.Notification) error {
	eventType, ok := typeMap[ntf.Type]
	if !ok {
		return fmt.Errorf("hangout notification: unknown type %q", ntf.Type)
	}

	// HangoutID has no dedicated field on notification.Event — it's
	// carried in Metadata, same as every other domain's trigger
	// point (activity_id, archive_id, badge_id, and so on).
	metadata, err := json.Marshal(map[string]string{
		"hangout_id": ntf.HangoutID.String(),
	})
	if err != nil {
		return err
	}

	return n.publish.Publish(ctx, notificationdomain.Event{
		Type:        eventType,
		RecipientID: ntf.RecipientID,
		ActorID:     ntf.ActorID,
		Metadata:    metadata,
	})
}
