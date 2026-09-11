package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestWeightsSumToOne(t *testing.T) {
	sum := InterestWeight + HistoryWeight + EngagementWeight +
		QualityWeight + SuggestionAcceptanceWeight

	if sum != 1.0 {
		t.Fatalf("weights sum to %v, want 1.0", sum)
	}
}

func TestSignals_Weighted(t *testing.T) {
	tests := []struct {
		name    string
		signals Signals
		want    float64
	}{
		{
			name:    "all zero gives zero",
			signals: Signals{},
			want:    0,
		},
		{
			name:    "only interest match gives interest weight",
			signals: Signals{InterestMatch: 1},
			want:    InterestWeight,
		},
		{
			name: "all signals at 1 gives 1",
			signals: Signals{
				InterestMatch:        1,
				HistoryAffinity:      1,
				Engagement:           1,
				Quality:              1,
				SuggestionAcceptance: 1,
			},
			want: 1,
		},
		{
			name: "neutral 0.5 fallback across all signals gives 0.5",
			signals: Signals{
				InterestMatch:        0.5,
				HistoryAffinity:      0.5,
				Engagement:           0.5,
				Quality:              0.5,
				SuggestionAcceptance: 0.5,
			},
			want: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.signals.Weighted()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewScore_SetsTotalFromWeighted(t *testing.T) {
	userID := uuid.New()
	activityID := uuid.New()
	signals := Signals{InterestMatch: 1}

	score := NewScore(userID, activityID, signals)

	if score.Total != signals.Weighted() {
		t.Errorf("Total = %v, want %v", score.Total, signals.Weighted())
	}
	if score.UserID != userID || score.ActivityID != activityID {
		t.Errorf("Score does not carry the given IDs correctly")
	}
	if score.GeneratedAt.IsZero() {
		t.Errorf("GeneratedAt should not be zero")
	}
}
