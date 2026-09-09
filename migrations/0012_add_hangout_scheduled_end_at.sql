-- +migrate Up

-- ============================================================
-- hangouts.scheduled_end_at: the hangout's planned end time.
-- Nullable, like scheduled_at — not every hangout has a fixed end
-- time yet. Used by the Archive module to open a 1-week window for
-- the post-hangout media upload prompt.
-- ============================================================
ALTER TABLE hangouts
    ADD COLUMN scheduled_end_at TIMESTAMPTZ;

-- +migrate Down

ALTER TABLE hangouts
    DROP COLUMN IF EXISTS scheduled_end_at;
