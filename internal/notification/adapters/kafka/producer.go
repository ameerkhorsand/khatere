package kafka

import (
	"context"
	"encoding/json"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/bLorax/khatere-backend/internal/notification/domain"
)

// Producer implements application.EventPublisher by writing one
// JSON message per event to the configured topic.
type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireOne,
			// kafka-go stopped auto-creating a missing topic on first
			// publish as of v0.4.31 — this must be set explicitly, or
			// every Publish silently fails with "Unknown Topic Or
			// Partition" until the topic is created some other way.
			AllowAutoTopicCreation: true,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, event domain.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(event.RecipientID.String()),
		Value: payload,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
