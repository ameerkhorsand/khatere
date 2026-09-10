-- +migrate Up

-- ============================================================
-- user_recommendation_cache: one row per (user, activity), holding
-- that activity's last-computed ranked score for that user.
--
-- Go struct (internal/recommendation/domain):
--   type Score struct {
--       UserID      uuid.UUID
--       ActivityID  uuid.UUID
--       Signals     Signals     // stored as signals JSONB below
--       Total       float64
--       GeneratedAt time.Time
--   }
--
-- signals is JSONB, not four separate columns, because the signal
-- set is expected to grow (e.g. a 5th signal later) — adding a key
-- to the JSON needs no migration, unlike adding a column. total_score
-- stays a real column because it's what every read sorts and filters
-- on; pulling a value out of JSONB for that would be slower and
-- couldn't be indexed as cleanly.
--
-- No version/deleted_at: this is a fully derived, disposable cache —
-- Step 5's refresh worker replaces rows wholesale, it never edits
-- one in place, so there's nothing to detect a concurrent edit
-- against and nothing a user could "undelete".
-- ============================================================
CREATE TABLE user_recommendation_cache (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    activity_id  UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    signals      JSONB NOT NULL DEFAULT '{}'::jsonb,
    total_score  NUMERIC(5,4) NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, activity_id)
);

-- Serves ReplaceForUser (all rows for one user) and TopForUser
-- (all rows for one user, ordered by total_score descending) in one
-- index — both of RecommendationRepository's methods hit this.
CREATE INDEX idx_recommendation_cache_user_score
    ON user_recommendation_cache(user_id, total_score DESC);

-- +migrate Down

DROP TABLE IF EXISTS user_recommendation_cache;
