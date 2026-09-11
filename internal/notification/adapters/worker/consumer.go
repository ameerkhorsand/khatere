// Package worker runs the background Kafka consumer that turns
// published notification events into persisted rows. This mirrors
// the recommendation domain's RefreshWorker in shape — a Start
// method that blocks until ctx is cancelled, run in its own
// goroutine from main.go — but this one is event-triggered by
// reading a Kafka topic, not a periodic ticker.
package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/bLorax/khatere-backend/internal/notification/domain"
)

type ConsumerWorker struct {
	reader *kafkago.Reader
	repo   domain.Repository
}

func NewConsumerWorker(brokers []string, topic string, repo domain.Repository) *ConsumerWorker {
	return &ConsumerWorker{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: "notification-consumer",
		}),
		repo: repo,
	}
}

// Start blocks, reading one message at a time, until ctx is
// cancelled. A message that fails to persist is logged and skipped,
// not retried — a lost notification is a minor annoyance, not worth
// blocking every notification behind it or crash-looping the
// worker.
func (w *ConsumerWorker) Start(ctx context.Context) {
	for {
		msg, err := w.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			log.Printf("[notification worker] read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var event domain.Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[notification worker] bad event payload: %v", err)
			continue
		}

		notification := &domain.Notification{
			ID:          uuid.New(),
			Type:        event.Type,
			RecipientID: event.RecipientID,
			ActorID:     event.ActorID,
			Metadata:    event.Metadata,
			CreatedAt:   time.Now().UTC(),
		}
		if err := w.repo.Create(ctx, notification); err != nil {
			log.Printf("[notification worker] failed to persist %s for %s: %v", event.Type, event.RecipientID, err)
		}
	}
}

func (w *ConsumerWorker) Close() error {
	return w.reader.Close()
}
