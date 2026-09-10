package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bLorax/khatere-backend/internal/recommendation/domain"
)

// SuggestionStatsRepository implements domain.SuggestionStatsRepository
// against user_suggestion_stats (migration 0021).
type SuggestionStatsRepository struct {
	pool *pgxpool.Pool
}

func NewSuggestionStatsRepository(pool *pgxpool.Pool) *SuggestionStatsRepository {
	return &SuggestionStatsRepository{pool: pool}
}

// RecordSuggested upserts one row per activity ID, bumping
// TimesSuggested and refreshing LastSuggestedAt. All rows are
// written in one transaction so a reader never sees a partial
// batch.
func (r *SuggestionStatsRepository) RecordSuggested(ctx context.Context, userID uuid.UUID, activityIDs []uuid.UUID) error {
	if len(activityIDs) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, activityID := range activityIDs {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_suggestion_stats
				(user_id, activity_id, times_suggested, times_accepted, last_suggested_at)
			VALUES ($1, $2, 1, 0, now())
			ON CONFLICT (user_id, activity_id) DO UPDATE
				SET times_suggested = user_suggestion_stats.times_suggested + 1,
					last_suggested_at = now()
		`, userID, activityID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// RecordAccepted upserts one row, bumping TimesAccepted. If the
// activity was never explicitly suggested first (e.g. the user
// found it another way), this still creates a row rather than
// erroring, so the acceptance is not lost.
func (r *SuggestionStatsRepository) RecordAccepted(ctx context.Context, userID, activityID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_suggestion_stats
			(user_id, activity_id, times_suggested, times_accepted, last_suggested_at)
		VALUES ($1, $2, 0, 1, NULL)
		ON CONFLICT (user_id, activity_id) DO UPDATE
			SET times_accepted = user_suggestion_stats.times_accepted + 1
	`, userID, activityID)
	return err
}

// Get returns the zero-value domain.SuggestionStats (TimesSuggested
// and TimesAccepted both 0), not an error, when no row exists —
// callers rely on SuggestionStats.AcceptanceRate()'s neutral-0.5
// fallback for that case.
func (r *SuggestionStatsRepository) Get(ctx context.Context, userID, activityID uuid.UUID) (domain.SuggestionStats, error) {
	var s domain.SuggestionStats
	s.UserID = userID
	s.ActivityID = activityID

	err := r.pool.QueryRow(ctx, `
		SELECT times_suggested, times_accepted, last_suggested_at
		FROM user_suggestion_stats
		WHERE user_id = $1 AND activity_id = $2
	`, userID, activityID).Scan(&s.TimesSuggested, &s.TimesAccepted, &s.LastSuggestedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s, nil
		}
		return domain.SuggestionStats{}, err
	}

	return s, nil
}
