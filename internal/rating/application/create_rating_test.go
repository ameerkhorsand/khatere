package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/google/uuid"
)

// --- fakes -------------------------------------------------------

type fakeRatingRepo struct {
	existing  *domain.Rating
	findErr   error
	upsertErr error
	upserted  *domain.Rating
}

func (f *fakeRatingRepo) Upsert(ctx context.Context, r *domain.Rating) error {
	f.upserted = r
	return f.upsertErr
}
func (f *fakeRatingRepo) FindByActivityAndUser(ctx context.Context, activityID, userID uuid.UUID) (*domain.Rating, error) {
	return f.existing, f.findErr
}
func (f *fakeRatingRepo) Summarize(ctx context.Context, activityID uuid.UUID) (domain.Summary, error) {
	return domain.Summary{}, nil
}

type fakeAttendanceChecker struct {
	attended bool
	err      error
}

func (f *fakeAttendanceChecker) Attended(ctx context.Context, activityID, userID uuid.UUID) (bool, error) {
	return f.attended, f.err
}

// fakeSummaryCache is a no-op cache by default — every test that
// doesn't care about caching still needs a valid SummaryCache to
// pass in, since it's a required constructor argument now.
type fakeSummaryCache struct {
	invalidateErr       error
	invalidateCalled    bool
	invalidatedActivity uuid.UUID
}

func (f *fakeSummaryCache) Get(ctx context.Context, activityID uuid.UUID) (domain.Summary, bool, error) {
	return domain.Summary{}, false, nil
}
func (f *fakeSummaryCache) Set(ctx context.Context, activityID uuid.UUID, summary domain.Summary) error {
	return nil
}
func (f *fakeSummaryCache) Invalidate(ctx context.Context, activityID uuid.UUID) error {
	f.invalidateCalled = true
	f.invalidatedActivity = activityID
	return f.invalidateErr
}

// --- tests ---------------------------------------------------------

func TestCreateRatingUseCase_Execute(t *testing.T) {
	t.Run("invalid score is rejected before any lookup", func(t *testing.T) {
		attendance := &fakeAttendanceChecker{attended: false} // would fail too, proving score is checked first
		uc := NewCreateRatingUseCase(&fakeRatingRepo{}, attendance, &fakeSummaryCache{})

		_, err := uc.Execute(context.Background(), CreateRatingInput{Score: 3.3})

		if !errors.Is(err, domain.ErrInvalidScore) {
			t.Errorf("got %v, want ErrInvalidScore", err)
		}
	})

	t.Run("non-attendee is rejected", func(t *testing.T) {
		uc := NewCreateRatingUseCase(&fakeRatingRepo{}, &fakeAttendanceChecker{attended: false}, &fakeSummaryCache{})

		_, err := uc.Execute(context.Background(), CreateRatingInput{Score: 4.5})

		if !errors.Is(err, domain.ErrNotAttendee) {
			t.Errorf("got %v, want ErrNotAttendee", err)
		}
	})

	t.Run("first rating gets a new ID", func(t *testing.T) {
		repo := &fakeRatingRepo{findErr: domain.ErrRatingNotFound}
		uc := NewCreateRatingUseCase(repo, &fakeAttendanceChecker{attended: true}, &fakeSummaryCache{})

		got, err := uc.Execute(context.Background(), CreateRatingInput{Score: 5, ActivityID: uuid.New(), UserID: uuid.New()})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID == uuid.Nil {
			t.Errorf("expected a generated ID")
		}
	})

	t.Run("second rating overwrites the existing one, keeping its ID", func(t *testing.T) {
		existing := &domain.Rating{ID: uuid.New(), Score: 2}
		repo := &fakeRatingRepo{existing: existing}
		uc := NewCreateRatingUseCase(repo, &fakeAttendanceChecker{attended: true}, &fakeSummaryCache{})

		got, err := uc.Execute(context.Background(), CreateRatingInput{Score: 4.5, ActivityID: uuid.New(), UserID: uuid.New()})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != existing.ID {
			t.Errorf("expected existing rating ID to be reused, got a new one")
		}
		if got.Score != 4.5 {
			t.Errorf("score = %v, want 4.5", got.Score)
		}
		if repo.upserted.ID != existing.ID {
			t.Errorf("upsert should carry the existing ID, not a new one")
		}
	})

	t.Run("success invalidates the cache for that activity", func(t *testing.T) {
		cache := &fakeSummaryCache{}
		activityID := uuid.New()
		uc := NewCreateRatingUseCase(&fakeRatingRepo{findErr: domain.ErrRatingNotFound}, &fakeAttendanceChecker{attended: true}, cache)

		_, err := uc.Execute(context.Background(), CreateRatingInput{Score: 3, ActivityID: activityID, UserID: uuid.New()})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cache.invalidateCalled {
			t.Errorf("expected the summary cache to be invalidated after a new rating")
		}
		if cache.invalidatedActivity != activityID {
			t.Errorf("invalidated activity = %v, want %v", cache.invalidatedActivity, activityID)
		}
	})

	t.Run("cache invalidation failure does not fail the request", func(t *testing.T) {
		cache := &fakeSummaryCache{invalidateErr: errors.New("redis unreachable")}
		uc := NewCreateRatingUseCase(&fakeRatingRepo{findErr: domain.ErrRatingNotFound}, &fakeAttendanceChecker{attended: true}, cache)

		_, err := uc.Execute(context.Background(), CreateRatingInput{Score: 3, ActivityID: uuid.New(), UserID: uuid.New()})

		if err != nil {
			t.Fatalf("rating creation must succeed even if cache invalidation fails, got: %v", err)
		}
	})
}
