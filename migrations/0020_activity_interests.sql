-- +migrate Up

-- ============================================================
-- activity_interests: which interests an activity is tagged with.
-- Mirrors user_interests (migration 0003) exactly, so the interest
-- match signal is a plain set-overlap between the two tables — no
-- special-casing needed on either side.
--
-- An activity can carry more than one interest tag (e.g. a beach
-- volleyball meetup might tag both "sports" and "outdoor"), so this
-- is a many-to-many join, same shape as user_interests.
-- ============================================================
CREATE TABLE activity_interests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    interest_id UUID NOT NULL REFERENCES interests(id) ON DELETE RESTRICT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (activity_id, interest_id)
);

CREATE INDEX idx_activity_interests_activity_id ON activity_interests(activity_id);
CREATE INDEX idx_activity_interests_interest_id ON activity_interests(interest_id);

-- +migrate Down

DROP TABLE IF EXISTS activity_interests;
