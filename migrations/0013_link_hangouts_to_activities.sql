-- +migrate Up

ALTER TABLE hangouts
    ADD COLUMN activity_id UUID REFERENCES activities(id) ON DELETE SET NULL;

CREATE INDEX idx_hangouts_activity_id ON hangouts(activity_id) WHERE activity_id IS NOT NULL;

-- +migrate Down

DROP INDEX IF EXISTS idx_hangouts_activity_id;
ALTER TABLE hangouts DROP COLUMN IF EXISTS activity_id;
