package domain

import (
	"time"

	"github.com/google/uuid"
)

// Signal weights. Each weight is the share a signal contributes to
// the final score. They must sum to 1.0 — see Signals.Weighted
// below, which enforces this in a unit test rather than at runtime,
// since these are compile-time constants.
const (
	InterestWeight             = 0.35
	HistoryWeight              = 0.20
	EngagementWeight           = 0.15
	QualityWeight              = 0.15
	SuggestionAcceptanceWeight = 0.15
)

// Signals holds the five raw inputs to the ranking formula, each
// normalized to the range [0, 1] by the use case that builds one
// (see application.BuildSignals). Domain code never computes these
// from raw rows itself — it only combines already-normalized values,
// keeping this package free of SQL-shaped knowledge.
type Signals struct {
	// InterestMatch is how well an activity matches the user's
	// stated/inferred interests (interest lookup table overlap).
	InterestMatch float64

	// HistoryAffinity is how often the user has engaged with this
	// activity or similar ones before (acceptance/attendance rate).
	HistoryAffinity float64

	// Engagement is the user's general in-app activity level
	// (circle size, hangouts organized/joined, recency of use).
	Engagement float64

	// Quality is the activity's own reputation: rating average and
	// comment-vote quality, both already 0..1 normalized.
	Quality float64

	// SuggestionAcceptance is how often this exact activity, once
	// shown to this user as a suggestion, actually turned into a
	// hangout they organized (user_suggestion_stats, Step 4). This
	// is distinct from HistoryAffinity: HistoryAffinity looks at
	// whether past hangouts for this activity finished well;
	// SuggestionAcceptance looks at whether being suggested this
	// activity leads to action at all, reinforcing recurring
	// interests (e.g. hiking) over time.
	SuggestionAcceptance float64
}

// Weighted combines the five signals into a single score in [0, 1]
// using the package-level weight constants.
func (s Signals) Weighted() float64 {
	return s.InterestMatch*InterestWeight +
		s.HistoryAffinity*HistoryWeight +
		s.Engagement*EngagementWeight +
		s.Quality*QualityWeight +
		s.SuggestionAcceptance*SuggestionAcceptanceWeight
}

// Score is one activity's ranked suggestion for one user. It is the
// unit stored in the recommendation cache and returned by the
// /recommendations endpoint.
type Score struct {
	UserID      uuid.UUID
	ActivityID  uuid.UUID
	Signals     Signals
	Total       float64
	GeneratedAt time.Time
}

// NewScore builds a Score from raw signals, computing Total via
// Signals.Weighted so callers never have to remember to call it.
func NewScore(userID, activityID uuid.UUID, signals Signals) Score {
	return Score{
		UserID:      userID,
		ActivityID:  activityID,
		Signals:     signals,
		Total:       signals.Weighted(),
		GeneratedAt: time.Now().UTC(),
	}
}
