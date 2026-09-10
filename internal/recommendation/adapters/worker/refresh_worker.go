// Package worker runs the periodic background refresh for the
// recommendation cache (Step 5). Nothing else in this repo has a
// worker-pool pattern to mirror yet, so this is a plain
// ticker-driven loop — the simplest thing that fits side A
// (periodic, not event-triggered) from the roadmap.
package worker

import (
	"context"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/recommendation/application"
	"github.com/bLorax/khatere-backend/internal/recommendation/domain"
)

// RefreshWorker wakes up every Interval and recomputes the
// recommendation cache for every user whose cache is older than
// StaleAfter.
type RefreshWorker struct {
	cache      domain.RecommendationRepository
	generate   *application.GenerateSuggestionsUseCase
	interval   time.Duration
	staleAfter time.Duration
}

func New(cache domain.RecommendationRepository, generate *application.GenerateSuggestionsUseCase, interval, staleAfter time.Duration) *RefreshWorker {
	return &RefreshWorker{
		cache:      cache,
		generate:   generate,
		interval:   interval,
		staleAfter: staleAfter,
	}
}

// Start blocks, running one refresh pass immediately and then again
// every Interval, until ctx is cancelled. Callers run this in its
// own goroutine (see main.go) and cancel ctx on shutdown.
func (w *RefreshWorker) Start(ctx context.Context) {
	w.runOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

// runOnce refreshes every stale user once. One user's failure is
// logged and skipped, not fatal to the run — the same best-effort
// reasoning used for notifications and archive creation elsewhere
// in this repo, since a background refresh is not something a user
// is waiting on.
func (w *RefreshWorker) runOnce(ctx context.Context) {
	cutoff := time.Now().UTC().Add(-w.staleAfter)

	userIDs, err := w.cache.StaleUserIDs(ctx, cutoff)
	if err != nil {
		log.Printf("recommendation worker: failed to list stale users: %v", err)
		return
	}

	for _, userID := range userIDs {
		if _, err := w.generate.Execute(ctx, userID); err != nil {
			log.Printf("recommendation worker: failed to refresh suggestions for user %s: %v", userID, err)
		}
	}
}
