package domain

import (
	"time"

	"github.com/google/uuid"
)

// SuggestionStats tracks, per user per activity, how often an
// activity was suggested versus accepted, so a recurring interest
// (e.g. hiking) gets reinforced over time. This is distinct from
// HistoryAffinity, which only looks at a user's own completed or
// cancelled hangouts for that one exact activity.
type SuggestionStats struct {
	UserID          uuid.UUID
	ActivityID      uuid.UUID
	TimesSuggested  int
	TimesAccepted   int
	LastSuggestedAt *time.Time
}

// AcceptanceRate returns the share of suggestions that were
// accepted, normalized to [0, 1]. Returns 0.5 (neutral) when the
// activity has never been suggested to this user, matching this
// domain's no-data-neutral rule (see Signals' doc comment in
// recommendation.go).
func (s SuggestionStats) AcceptanceRate() float64 {
	if s.TimesSuggested == 0 {
		return 0.5
	}
	return float64(s.TimesAccepted) / float64(s.TimesSuggested)
}
