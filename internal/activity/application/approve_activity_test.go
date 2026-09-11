package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bLorax/khatere-backend/internal/activity/domain"
	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeActivityRepo struct {
	activity  *domain.Activity
	findErr   error
	updateErr error
	updated   *domain.Activity // captured so a test can assert on the write
}

func (f *fakeActivityRepo) Create(ctx context.Context, a *domain.Activity) error { return nil }
func (f *fakeActivityRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Activity, error) {
	return f.activity, f.findErr
}
func (f *fakeActivityRepo) List(ctx context.Context, filter domain.ListFilter) ([]domain.Activity, error) {
	return nil, nil
}
func (f *fakeActivityRepo) ListPendingQueue(ctx context.Context) ([]domain.Activity, error) {
	return nil, nil
}
func (f *fakeActivityRepo) Update(ctx context.Context, a *domain.Activity) error {
	f.updated = a
	return f.updateErr
}

type fakeNotificationPublisher struct {
	publishErr error
	published  *notificationdomain.Event
}

func (f *fakeNotificationPublisher) Publish(ctx context.Context, e notificationdomain.Event) error {
	f.published = &e
	return f.publishErr
}

// --- tests ---------------------------------------------------------

func TestApproveActivityUseCase_Execute(t *testing.T) {
	t.Run("non-pending activity is rejected", func(t *testing.T) {
		activity := &domain.Activity{ID: uuid.New(), Status: domain.ActivityStatusApproved}
		repo := &fakeActivityRepo{activity: activity}
		notify := &fakeNotificationPublisher{}
		uc := NewApproveActivityUseCase(repo, notify)

		_, err := uc.Execute(context.Background(), ApproveActivityInput{ActivityID: activity.ID})

		if !errors.Is(err, domain.ErrActivityNotPending) {
			t.Errorf("got %v, want ErrActivityNotPending", err)
		}
	})

	t.Run("update failure is passed through", func(t *testing.T) {
		activity := &domain.Activity{ID: uuid.New(), Status: domain.ActivityStatusPending}
		repo := &fakeActivityRepo{activity: activity, updateErr: errors.New("version conflict")}
		notify := &fakeNotificationPublisher{}
		uc := NewApproveActivityUseCase(repo, notify)

		_, err := uc.Execute(context.Background(), ApproveActivityInput{ActivityID: activity.ID})

		if err == nil || err.Error() != "version conflict" {
			t.Errorf("got %v, want update error to be returned", err)
		}
	})

	t.Run("notification failure does not fail the approval", func(t *testing.T) {
		activity := &domain.Activity{ID: uuid.New(), Status: domain.ActivityStatusPending, CreatedBy: uuid.New()}
		repo := &fakeActivityRepo{activity: activity}
		notify := &fakeNotificationPublisher{publishErr: errors.New("kafka unreachable")}
		uc := NewApproveActivityUseCase(repo, notify)

		got, err := uc.Execute(context.Background(), ApproveActivityInput{ActivityID: activity.ID})

		if err != nil {
			t.Fatalf("approval must succeed even if the notification fails, got: %v", err)
		}
		if got.Status != domain.ActivityStatusApproved {
			t.Errorf("status = %v, want approved", got.Status)
		}
	})

	t.Run("success sets status, reviewer, and reviewed time", func(t *testing.T) {
		activity := &domain.Activity{ID: uuid.New(), Status: domain.ActivityStatusPending, CreatedBy: uuid.New()}
		repo := &fakeActivityRepo{activity: activity}
		notify := &fakeNotificationPublisher{}
		moderatorID := uuid.New()
		uc := NewApproveActivityUseCase(repo, notify)

		got, err := uc.Execute(context.Background(), ApproveActivityInput{ActivityID: activity.ID, ModeratorID: moderatorID})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != domain.ActivityStatusApproved {
			t.Errorf("status = %v, want approved", got.Status)
		}
		if got.ReviewedBy == nil || *got.ReviewedBy != moderatorID {
			t.Errorf("ReviewedBy not set to moderator")
		}
		if got.ReviewedAt == nil {
			t.Errorf("ReviewedAt should be set")
		}
		if repo.updated == nil {
			t.Errorf("expected Update to be called")
		}
		if notify.published == nil || notify.published.Type != notificationdomain.TypeActivityApproved {
			t.Errorf("expected a TypeActivityApproved notification to be published")
		}
	})
}
