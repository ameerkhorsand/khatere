-- +migrate Up

-- ============================================================
-- user_suggestion_stats: per-user, per-activity counters that track
-- how often an activity was suggested to a user versus actually
-- accepted (the user organized or joined a hangout around it).
--
-- This is separate from HistoryAffinity (which only looks at a
-- user's own completed/cancelled hangouts for one exact activity).
-- This table lets a recurring interest (e.g. hiking) get reinforced
-- over time, even before the user has ever attended that activity.
--
-- Disposable/derived row — no soft delete, no version column, same
-- reasoning as user_recommendation_cache (migration 0019).
-- ============================================================
CREATE TABLE user_suggestion_stats (
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_id        UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    times_suggested    INTEGER NOT NULL DEFAULT 0,
    times_accepted     INTEGER NOT NULL DEFAULT 0,
    last_suggested_at  TIMESTAMPTZ,
    PRIMARY KEY (user_id, activity_id)
);

-- +migrate Down

DROP TABLE IF EXISTS user_suggestion_stats;
