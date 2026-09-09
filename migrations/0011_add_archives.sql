-- +migrate Up

-- ============================================================
-- archives: a snapshot of one resolved hangout (chat log + media).
-- Created once, when the hangout's status becomes 'completed' or
-- 'cancelled'. Status turns to 'purged' once every participant of
-- the underlying hangout has left a deletion mark; the row is kept
-- for audit but its media and chat_snapshot are cleared by the
-- application layer at that point.
--
-- Go struct (internal/archive/domain):
--   type Archive struct {
--       ID           uuid.UUID
--       HangoutID    uuid.UUID
--       ChatSnapshot string
--       Status       ArchiveStatus // enum: "active" | "purged"
--       CreatedAt    time.Time
--       UpdatedAt    time.Time
--   }
-- ============================================================
CREATE TABLE archives (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hangout_id     UUID NOT NULL REFERENCES hangouts(id) ON DELETE CASCADE,
    chat_snapshot  TEXT NOT NULL DEFAULT '',
    status         VARCHAR(20) NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active', 'purged')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hangout_id)
);

CREATE INDEX idx_archives_hangout_id ON archives(hangout_id);
CREATE INDEX idx_archives_status ON archives(status);

-- No update trigger here, same reason as hangout_meetup_pin_confirmations
-- in 0009: this table has no version column, so
-- set_updated_at_and_bump_version() would fail on every update.
-- The Go repository sets updated_at by hand instead.

-- ============================================================
-- archive_media: one row per uploaded file attached to an archive.
-- duration_seconds is set for 'video' and 'audio' only, and is
-- capped at 60 by the application layer before the row is written.
-- deleted_at is a soft delete, kept for the same reason as
-- hangout_messages.deleted_at.
--
-- Go struct (internal/archive/domain):
--   type ArchiveMedia struct {
--       ID              uuid.UUID
--       ArchiveID       uuid.UUID
--       UploaderID      uuid.UUID
--       MediaType       MediaType // enum: "photo" | "gif" | "video" | "audio"
--       StorageKey      string
--       DurationSeconds *int
--       CreatedAt       time.Time
--       DeletedAt       *time.Time
--   }
-- ============================================================
CREATE TABLE archive_media (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    archive_id        UUID NOT NULL REFERENCES archives(id) ON DELETE CASCADE,
    uploader_id       UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    media_type        VARCHAR(20) NOT NULL
                      CHECK (media_type IN ('photo', 'gif', 'video', 'audio')),
    storage_key       TEXT NOT NULL,
    duration_seconds  INTEGER
                      CHECK (duration_seconds IS NULL OR duration_seconds BETWEEN 1 AND 60),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX idx_archive_media_archive_id ON archive_media(archive_id);
CREATE INDEX idx_archive_media_uploader_id ON archive_media(uploader_id);

-- ============================================================
-- archive_deletion_marks: one row per participant who asked to
-- delete an archive. Personal delete only — it hides the archive
-- for that user. The application layer purges the archive on the
-- server once the row count here equals the hangout's participant
-- count.
--
-- Go struct (internal/archive/domain):
--   type ArchiveDeletionMark struct {
--       ArchiveID uuid.UUID
--       UserID    uuid.UUID
--       CreatedAt time.Time
--   }
-- No id column — (archive_id, user_id) is the natural key, and a
-- mark is never updated, only inserted or counted.
-- ============================================================
CREATE TABLE archive_deletion_marks (
    archive_id  UUID NOT NULL REFERENCES archives(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (archive_id, user_id)
);

CREATE INDEX idx_archive_deletion_marks_archive_id ON archive_deletion_marks(archive_id);

-- +migrate Down

DROP TABLE IF EXISTS archive_deletion_marks;
DROP TABLE IF EXISTS archive_media;
DROP TABLE IF EXISTS archives;
