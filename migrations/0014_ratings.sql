-- +migrate Up

-- ============================================================
-- ratings: one row per user per activity. Only a user who
-- attended a completed hangout linked to the activity may rate
-- it (checked in the application layer, not the database).
--
-- Go struct (internal/rating/domain):
--   type Rating struct {
--       ID         uuid.UUID
--       ActivityID uuid.UUID
--       UserID     uuid.UUID   // FK to accounts.id
--       Score      float64     // 0.0 to 5.0, in steps of 0.5
--       CreatedAt  time.Time
--       UpdatedAt  time.Time
--   }
-- No version/deleted_at: a rating is a small, disposable row,
-- same reasoning as hangout_participants — a user edits it in
-- place, and a removed rating is just deleted, not soft-deleted.
-- ============================================================
CREATE TABLE ratings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id  UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    score        NUMERIC(2,1) NOT NULL
                 CHECK (score >= 0 AND score <= 5 AND (score * 2) = FLOOR(score * 2)),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (activity_id, user_id)
);

CREATE INDEX idx_ratings_activity_id ON ratings(activity_id);
CREATE INDEX idx_ratings_user_id ON ratings(user_id);

-- No trigger here, same reason as archives in 0011: no version
-- column, so set_updated_at_and_bump_version() would fail. The
-- Go repository sets updated_at by hand on upsert.

-- +migrate Down

DROP TABLE IF EXISTS ratings;
