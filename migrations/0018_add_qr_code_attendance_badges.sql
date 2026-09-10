-- +migrate Up

-- ============================================================
-- qr_codes: one fixed QR code per activity. A host cannot make
-- a second QR code for the same activity. The code value does
-- not change after creation.
--
-- Go struct (internal/qrcode/domain):
--   type QRCode struct {
--       ID         uuid.UUID
--       ActivityID uuid.UUID
--       Code       string      // random opaque token, encoded in the QR image
--       CreatedAt  time.Time
--   }
-- No version/updated_at: the row is never edited after creation.
-- ============================================================
CREATE TABLE qr_codes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id  UUID NOT NULL UNIQUE REFERENCES activities(id) ON DELETE CASCADE,
    code         TEXT NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_qr_codes_code ON qr_codes(code);

-- ============================================================
-- badges: one badge per activity, set up by hand by the host.
-- The in-app badge design tool is deferred, so a host fills in
-- these fields directly (through a moderator/admin-assisted
-- flow for now).
--
-- Go struct (internal/badge/domain):
--   type Badge struct {
--       ID         uuid.UUID
--       ActivityID uuid.UUID
--       CreatedBy  uuid.UUID   // FK to accounts.id (the host)
--       Name       string
--       IconKey    string      // object storage key, not a URL — same
--                               // pattern as User.ProfilePictureKey.
--                               // Generate a presigned URL on read.
--       Version    int
--       CreatedAt  time.Time
--       UpdatedAt  time.Time
--   }
-- No deleted_at: user_badges keeps its own copy of the name and
-- icon key at award time, so a later badge edit or removal never
-- changes what a user already earned. See user_badges below.
-- ============================================================
CREATE TABLE badges (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id  UUID NOT NULL UNIQUE REFERENCES activities(id) ON DELETE CASCADE,
    created_by   UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    icon_key     TEXT,
    version      INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_badges_updated_at
    BEFORE UPDATE ON badges
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- attendance_verifications: one row per user per activity scan.
-- A scan of the activity's QR code makes this row. Same shape
-- reasoning as ratings (0014): a small row, edited or removed
-- outright, never soft-deleted.
--
-- Go struct (internal/attendance/domain):
--   type AttendanceVerification struct {
--       ID          uuid.UUID
--       ActivityID  uuid.UUID
--       UserID      uuid.UUID   // FK to accounts.id
--       QRCodeID    uuid.UUID
--       VerifiedAt  time.Time
--   }
-- ============================================================
CREATE TABLE attendance_verifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id  UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    qr_code_id   UUID NOT NULL REFERENCES qr_codes(id) ON DELETE CASCADE,
    verified_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (activity_id, user_id)
);

CREATE INDEX idx_attendance_verifications_activity_id ON attendance_verifications(activity_id);
CREATE INDEX idx_attendance_verifications_user_id ON attendance_verifications(user_id);

-- ============================================================
-- user_badges: one row per badge earned by a user. Made right
-- after a verified scan, when the activity has a badge set up.
--
-- name_snapshot / icon_key_snapshot hold a copy of the badge's
-- fields at award time. This is why a user keeps a badge on
-- their profile even if the host later edits the badge, removes
-- the badge, or removes the activity itself: badge_id and
-- activity_id can go to NULL, but the snapshot fields stay.
--
-- icon_key_snapshot is an object storage key, same pattern as
-- badges.icon_key — generate a presigned URL on read.
--
-- Go struct (internal/userbadge/domain):
--   type UserBadge struct {
--       ID                uuid.UUID
--       UserID            uuid.UUID   // FK to accounts.id
--       BadgeID           *uuid.UUID  // nil if the badge was later removed
--       ActivityID        *uuid.UUID  // nil if the activity was later removed
--       NameSnapshot      string
--       IconKeySnapshot   string
--       Visible           bool        // user's own show/hide choice
--       AwardedAt         time.Time
--   }
-- ============================================================
CREATE TABLE user_badges (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    badge_id           UUID REFERENCES badges(id) ON DELETE SET NULL,
    activity_id        UUID REFERENCES activities(id) ON DELETE SET NULL,
    name_snapshot      VARCHAR(100) NOT NULL,
    icon_key_snapshot  TEXT,
    visible            BOOLEAN NOT NULL DEFAULT true,
    awarded_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, badge_id)
);

CREATE INDEX idx_user_badges_user_id ON user_badges(user_id);

-- +migrate Down

DROP TABLE IF EXISTS user_badges;
DROP TABLE IF EXISTS attendance_verifications;
DROP TABLE IF EXISTS badges;
DROP TABLE IF EXISTS qr_codes;
