-- +migrate Up

CREATE TABLE activities (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title             VARCHAR(150) NOT NULL,
    description       TEXT,
    source_type       VARCHAR(20) NOT NULL CHECK (source_type IN ('moderator', 'host', 'user')),
    created_by        UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    status            VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by       UUID REFERENCES accounts(id) ON DELETE SET NULL,
    reviewed_at       TIMESTAMPTZ,
    rejection_reason  TEXT,
    metadata          JSONB NOT NULL DEFAULT '{}',
    version           INTEGER NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX idx_activities_status ON activities(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_activities_source_type ON activities(source_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_activities_created_by ON activities(created_by);
CREATE INDEX idx_activities_deleted_at ON activities(deleted_at) WHERE deleted_at IS NOT NULL;

-- Fast lookup of the pending moderation queue specifically.
CREATE INDEX idx_activities_pending_queue ON activities(created_at) WHERE status = 'pending' AND deleted_at IS NULL;

CREATE TRIGGER trg_activities_updated_at
    BEFORE UPDATE ON activities
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- +migrate Down

DROP TRIGGER IF EXISTS trg_activities_updated_at ON activities;
DROP TABLE IF EXISTS activities;
