package domain

import (
	"context"

	"github.com/google/uuid"
)

// Summary is the aggregate shown on an activity: the average score
// and how many ratings it rests on. Average is 0 when Count is 0 —
// callers must check Count before treating Average as meaningful.
type Summary struct {
	Average float64
	Count   int
}

type RatingRepository interface {
	// Upsert inserts a new rating, or overwrites the existing
	// score for (ActivityID, UserID) if one already exists. A
	// user can only ever have one rating per activity — see the
	// UNIQUE constraint in migration 0014.
	Upsert(ctx context.Context, rating *Rating) error

	FindByActivityAndUser(ctx context.Context, activityID, userID uuid.UUID) (*Rating, error)

	// Summarize returns the average score and count for an
	// activity. Count is 0, Average is 0 when nobody has rated it
	// yet — this is not an error.
	Summarize(ctx context.Context, activityID uuid.UUID) (Summary, error)
}
