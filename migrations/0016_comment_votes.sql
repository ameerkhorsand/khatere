-- +migrate Up

-- ============================================================
-- comment_votes: one row per user per comment. Feeds the ranking
-- formula in Phase 6 (70% vote weight); a verification boost
-- column is added on top of this in Phase 7.
--
-- Go struct (internal/comment/domain):
--   type CommentVote struct {
--       CommentID uuid.UUID
--       UserID    uuid.UUID   // FK to accounts.id
--       Value     int         // 1 = up, -1 = down
--       CreatedAt time.Time
--   }
-- No id, no version, no deleted_at: (comment_id, user_id) is the
-- natural key, and a vote is replaced in place, not soft-deleted —
-- same reasoning as archive_deletion_marks in 0011.
-- ============================================================
CREATE TABLE comment_votes (
    comment_id  UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    value       SMALLINT NOT NULL CHECK (value IN (1, -1)),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (comment_id, user_id)
);

CREATE INDEX idx_comment_votes_comment_id ON comment_votes(comment_id);

-- +migrate Down

DROP TABLE IF EXISTS comment_votes;
