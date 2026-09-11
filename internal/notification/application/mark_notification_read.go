package application

import (
	"context"

	"github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

// MarkNotificationReadUseCase marks one notification read. The
// repository's MarkRead already scopes the update to recipientID
// (see domain.Repository), so a caller can never mark someone
// else's notification read — this use case adds no extra check on
// top of that.
type MarkNotificationReadUseCase struct {
	notifications domain.Repository
}

func NewMarkNotificationReadUseCase(notifications domain.Repository) *MarkNotificationReadUseCase {
	return &MarkNotificationReadUseCase{notifications: notifications}
}

func (uc *MarkNotificationReadUseCase) Execute(ctx context.Context, notificationID, recipientID uuid.UUID) error {
	return uc.notifications.MarkRead(ctx, notificationID, recipientID)
}
