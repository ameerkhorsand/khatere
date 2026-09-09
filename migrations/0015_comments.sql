-- +migrate Up

-- ============================================================
-- comments: a review left by any user on an activity. Gated
-- behind moderation, same status flow as activities. Any user
-- may comment (not attendee-only, unlike ratings).
--
-- Go struct (internal/comment/domain):
--   type Comment struct {
--       ID          uuid.UUID
--       ActivityID  uuid.UUID
--       UserID      uuid.UUID     // FK to accounts.id
--       Body        string
--       Status      CommentStatus // enum: "pending" | "approved" | "rejected"
--       ReviewedBy  *uuid.UUID
--       ReviewedAt  *time.Time
--       Metadata    map[string]any
--       Version     int
--       CreatedAt   time.Time
--       UpdatedAt   time.Time
--       DeletedAt   *time.Time
--   }
-- ============================================================
CREATE TABLE comments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id   UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    body          TEXT NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by   UUID REFERENCES accounts(id) ON DELETE SET NULL,
    reviewed_at   TIMESTAMPTZ,
    metadata      JSONB NOT NULL DEFAULT '{}',
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_comments_activity_id ON comments(activity_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_status ON comments(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_deleted_at ON comments(deleted_at) WHERE deleted_at IS NOT NULL;

-- Fast lookup of the pending moderation queue, same as
-- idx_activities_pending_queue in 0007.
CREATE INDEX idx_comments_pending_queue ON comments(created_at) WHERE status = 'pending' AND deleted_at IS NULL;

CREATE TRIGGER trg_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- +migrate Down

DROP TRIGGER IF EXISTS trg_comments_updated_at ON comments;
DROP TABLE IF EXISTS comments;
