package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/rating/domain"
	"github.com/google/uuid"
)

// AttendanceChecker is intentionally small so the rating use case
// doesn't depend on the concrete hangout repository — same pattern
// as UserFinder in the circle HTTP adapter. HangoutRepository.Attended
// (internal/hangout/adapters/postgres) satisfies this interface.
type AttendanceChecker interface {
	// Attended reports whether userID has an 'accepted' invite
	// status on a hangout that is (a) linked to activityID via
	// hangouts.activity_id, and (b) status = 'completed'.
	Attended(ctx context.Context, activityID, userID uuid.UUID) (bool, error)
}

type CreateRatingUseCase struct {
	ratings    domain.RatingRepository
	attendance AttendanceChecker
}

func NewCreateRatingUseCase(ratings domain.RatingRepository, attendance AttendanceChecker) *CreateRatingUseCase {
	return &CreateRatingUseCase{ratings: ratings, attendance: attendance}
}

type CreateRatingInput struct {
	ActivityID uuid.UUID
	UserID     uuid.UUID
	Score      float64
}

func (uc *CreateRatingUseCase) Execute(ctx context.Context, in CreateRatingInput) (*domain.Rating, error) {
	if !domain.Valid(in.Score) {
		return nil, domain.ErrInvalidScore
	}

	attended, err := uc.attendance.Attended(ctx, in.ActivityID, in.UserID)
	if err != nil {
		return nil, err
	}
	if !attended {
		return nil, domain.ErrNotAttendee
	}

	now := time.Now()
	existing, err := uc.ratings.FindByActivityAndUser(ctx, in.ActivityID, in.UserID)
	if err != nil && err != domain.ErrRatingNotFound {
		return nil, err
	}

	rating := &domain.Rating{
		ID:         uuid.New(),
		ActivityID: in.ActivityID,
		UserID:     in.UserID,
		Score:      in.Score,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if existing != nil {
		// A second rating from the same user overwrites the
		// first (UNIQUE(activity_id, user_id) in migration
		// 0014) rather than erroring — matches "the user can
		// edit it in place" from the migration's comment.
		rating.ID = existing.ID
		rating.CreatedAt = existing.CreatedAt
	}

	if err := uc.ratings.Upsert(ctx, rating); err != nil {
		return nil, err
	}

	return rating, nil
}
