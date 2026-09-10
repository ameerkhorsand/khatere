package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/bLorax/khatere-backend/internal/recommendation/domain"
)

// DefaultLimit is how many suggestions are returned when the caller
// doesn't specify a limit.
const DefaultLimit = 20

// GetSuggestionsUseCase reads a user's ranked suggestions from the
// cache. On a cache miss — a new user, or a user whose cache hasn't
// been populated yet by the background refresh worker (Step 5) — it
// falls back to computing live rather than returning an empty feed.
type GetSuggestionsUseCase struct {
	cache           domain.RecommendationRepository
	generate        *GenerateSuggestionsUseCase
	suggestionStats domain.SuggestionStatsRepository
}

func NewGetSuggestionsUseCase(cache domain.RecommendationRepository, generate *GenerateSuggestionsUseCase, suggestionStats domain.SuggestionStatsRepository) *GetSuggestionsUseCase {
	return &GetSuggestionsUseCase{cache: cache, generate: generate, suggestionStats: suggestionStats}
}

func (uc *GetSuggestionsUseCase) Execute(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Score, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}

	cached, err := uc.cache.TopForUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	if len(cached) > 0 {
		if err := uc.recordSuggested(ctx, userID, cached); err != nil {
			return nil, err
		}
		return cached, nil
	}

	fresh, err := uc.generate.Execute(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(fresh) > limit {
		fresh = fresh[:limit]
	}
	if err := uc.recordSuggested(ctx, userID, fresh); err != nil {
		return nil, err
	}
	return fresh, nil
}

// recordSuggested tells the suggestion-stats table that every
// activity in scores was just shown to userID. This runs on every
// call — cache hit or cache miss — so times_suggested reflects what
// the user actually saw, not just what was computed.
func (uc *GetSuggestionsUseCase) recordSuggested(ctx context.Context, userID uuid.UUID, scores []domain.Score) error {
	activityIDs := make([]uuid.UUID, len(scores))
	for i, score := range scores {
		activityIDs[i] = score.ActivityID
	}
	return uc.suggestionStats.RecordSuggested(ctx, userID, activityIDs)
}
