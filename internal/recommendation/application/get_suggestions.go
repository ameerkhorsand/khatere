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
	activities      ActivityLookup
}

func NewGetSuggestionsUseCase(cache domain.RecommendationRepository, generate *GenerateSuggestionsUseCase, suggestionStats domain.SuggestionStatsRepository, activities ActivityLookup) *GetSuggestionsUseCase {
	return &GetSuggestionsUseCase{cache: cache, generate: generate, suggestionStats: suggestionStats, activities: activities}
}

func (uc *GetSuggestionsUseCase) Execute(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Suggestion, error) {
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
		return uc.EnrichWithActivities(ctx, cached)
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
	return uc.EnrichWithActivities(ctx, fresh)
}

// EnrichWithActivities joins each score with its activity's display
// data in one batched lookup. It is exported so RefreshSuggestions
// (which calls GenerateSuggestionsUseCase directly, bypassing
// Execute above) can reuse the same join instead of duplicating it.
// A score whose activity has since been deleted or rejected is left
// out of the result entirely, rather than returned with a blank
// activity — the frontend gets a shorter, always-valid list instead
// of having to handle a null activity field itself.
func (uc *GetSuggestionsUseCase) EnrichWithActivities(ctx context.Context, scores []domain.Score) ([]domain.Suggestion, error) {
	ids := make([]uuid.UUID, len(scores))
	for i, score := range scores {
		ids[i] = score.ActivityID
	}

	summaries, err := uc.activities.ActivitySummaries(ctx, ids)
	if err != nil {
		return nil, err
	}

	suggestions := make([]domain.Suggestion, 0, len(scores))
	for _, score := range scores {
		summary, ok := summaries[score.ActivityID]
		if !ok {
			continue
		}
		suggestions = append(suggestions, domain.Suggestion{Score: score, Activity: summary})
	}
	return suggestions, nil
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
