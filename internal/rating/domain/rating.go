package domain

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

// MinScore and MaxScore bound a rating. Steps of 0.5 only — see
// Valid below. Enforced again in the database via a CHECK
// constraint (migration 0014), so this check is a fast, friendly
// rejection before the write even happens.
const (
	MinScore = 0.0
	MaxScore = 5.0
)

type Rating struct {
	ID         uuid.UUID
	ActivityID uuid.UUID
	UserID     uuid.UUID
	Score      float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Valid reports whether score is within [MinScore, MaxScore] and
// lands on a 0.5 step (0, 0.5, 1, 1.5, ... 5).
func Valid(score float64) bool {
	if score < MinScore || score > MaxScore {
		return false
	}
	doubled := score * 2
	return math.Abs(doubled-math.Round(doubled)) < 1e-9
}

var (
	ErrRatingNotFound = errors.New("rating not found")
	ErrInvalidScore   = errors.New("score must be between 0 and 5, in steps of 0.5")
	ErrNotAttendee    = errors.New("only users who attended a completed hangout for this activity may rate it")
)
