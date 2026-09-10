package application

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/bLorax/khatere-backend/internal/recommendation/domain"
)

// ActivityCandidateSource lists the activities eligible to be scored
// for a user (visible, approved, not already part of a solo
// hangout they've already run, etc. — filtering rules live in the
// adapter). It is defined here rather than in ports.go because,
// unlike the four signal ports, nothing else in this domain needs
// it.
type ActivityCandidateSource interface {
	CandidateActivities(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// GenerateSuggestionsUseCase computes a ranked Score for every
// candidate activity a user hasn't been filtered out of, then
// replaces that user's cached suggestions with the fresh set.
type GenerateSuggestionsUseCase struct {
	candidates      ActivityCandidateSource
	interests       InterestSource
	history         HistorySource
	engagement      EngagementSource
	quality         QualitySource
	suggestionStats domain.SuggestionStatsRepository
	cache           domain.RecommendationRepository
}

func NewGenerateSuggestionsUseCase(
	candidates ActivityCandidateSource,
	interests InterestSource,
	history HistorySource,
	engagement EngagementSource,
	quality QualitySource,
	suggestionStats domain.SuggestionStatsRepository,
	cache domain.RecommendationRepository,
) *GenerateSuggestionsUseCase {
	return &GenerateSuggestionsUseCase{
		candidates:      candidates,
		interests:       interests,
		history:         history,
		engagement:      engagement,
		quality:         quality,
		suggestionStats: suggestionStats,
		cache:           cache,
	}
}

// Execute scores every candidate activity for userID, writes the
// result to the cache, and also returns it — the HTTP handler can
// use the return value directly on a cold cache instead of writing
// then immediately reading it back.
func (uc *GenerateSuggestionsUseCase) Execute(ctx context.Context, userID uuid.UUID) ([]domain.Score, error) {
	activityIDs, err := uc.candidates.CandidateActivities(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Engagement doesn't vary per activity, so it's fetched once per
	// run rather than once per candidate.
	engagementScore, err := uc.engagement.Engagement(ctx, userID)
	if err != nil {
		return nil, err
	}

	scores := make([]domain.Score, 0, len(activityIDs))
	for _, activityID := range activityIDs {
		interestScore, err := uc.interests.InterestMatch(ctx, userID, activityID)
		if err != nil {
			return nil, err
		}

		historyScore, err := uc.history.HistoryAffinity(ctx, userID, activityID)
		if err != nil {
			return nil, err
		}

		qualityScore, err := uc.quality.Quality(ctx, activityID)
		if err != nil {
			return nil, err
		}

		stats, err := uc.suggestionStats.Get(ctx, userID, activityID)
		if err != nil {
			return nil, err
		}

		signals := domain.Signals{
			InterestMatch:        interestScore,
			HistoryAffinity:      historyScore,
			Engagement:           engagementScore,
			Quality:              qualityScore,
			SuggestionAcceptance: stats.AcceptanceRate(),
		}

		scores = append(scores, domain.NewScore(userID, activityID, signals))
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Total > scores[j].Total
	})

	if err := uc.cache.ReplaceForUser(ctx, userID, scores); err != nil {
		return nil, err
	}

	return scores, nil
}
