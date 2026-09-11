package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

type fakeNotificationRepo struct {
	markReadErr    error
	markReadCalled bool
	notifID        uuid.UUID
	recipientID    uuid.UUID
}

func (f *fakeNotificationRepo) Create(ctx context.Context, n *domain.Notification) error { return nil }
func (f *fakeNotificationRepo) ListForRecipient(ctx context.Context, recipientID uuid.UUID, limit int) ([]domain.Notification, error) {
	return nil, nil
}
func (f *fakeNotificationRepo) MarkRead(ctx context.Context, notificationID, recipientID uuid.UUID) error {
	f.markReadCalled = true
	f.notifID = notificationID
	f.recipientID = recipientID
	return f.markReadErr
}

func TestMarkNotificationReadUseCase_Execute(t *testing.T) {
	t.Run("passes ids through and returns nil on success", func(t *testing.T) {
		repo := &fakeNotificationRepo{}
		uc := NewMarkNotificationReadUseCase(repo)
		notifID, recipientID := uuid.New(), uuid.New()

		err := uc.Execute(context.Background(), notifID, recipientID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !repo.markReadCalled {
			t.Fatalf("expected MarkRead to be called")
		}
		if repo.notifID != notifID || repo.recipientID != recipientID {
			t.Errorf("MarkRead called with wrong ids: got (%v, %v), want (%v, %v)",
				repo.notifID, repo.recipientID, notifID, recipientID)
		}
	})

	t.Run("repository error is passed through", func(t *testing.T) {
		repo := &fakeNotificationRepo{markReadErr: errors.New("not found or not owned")}
		uc := NewMarkNotificationReadUseCase(repo)

		err := uc.Execute(context.Background(), uuid.New(), uuid.New())

		if err == nil || err.Error() != "not found or not owned" {
			t.Errorf("got %v, want the repository error passed through", err)
		}
	})
}
