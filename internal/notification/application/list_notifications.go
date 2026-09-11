package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

// DefaultLimit is how many notifications are returned when the
// caller doesn't specify a limit.
const DefaultLimit = 20

// ListNotificationsUseCase returns one recipient's own notifications,
// newest first. This is a thin read on top of domain.Repository —
// all the real work (writing rows) happens in the consumer worker
// from Step 8.
type ListNotificationsUseCase struct {
	notifications domain.Repository
}

func NewListNotificationsUseCase(notifications domain.Repository) *ListNotificationsUseCase {
	return &ListNotificationsUseCase{notifications: notifications}
}

func (uc *ListNotificationsUseCase) Execute(ctx context.Context, recipientID uuid.UUID, limit int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	return uc.notifications.ListForRecipient(ctx, recipientID, limit)
}
