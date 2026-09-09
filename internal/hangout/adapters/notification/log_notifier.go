package notification

import (
	"context"
	"log"

	"github.com/bLorax/khatere-backend/internal/hangout/domain"
)

// LogNotifier writes each notification to the server log. It exists
// so the rest of the Hangout code can call domain.Notifier today.
// Replace it with a real adapter (email, push, a notifications
// table) once that infrastructure exists — no other file needs to
// change, since every caller only depends on the domain.Notifier
// interface.
type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) Notify(ctx context.Context, ntf domain.Notification) error {
	log.Printf(
		"[hangout notification] type=%s recipient=%s actor=%s hangout=%s",
		ntf.Type, ntf.RecipientID, ntf.ActorID, ntf.HangoutID,
	)
	return nil
}
