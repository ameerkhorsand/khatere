-- +migrate Up

-- ============================================================
-- hangouts: a planned meetup between an organizer and invited
-- participants. Not moderated (unlike activities); the organizer
-- controls the lifecycle directly.
--
-- Go struct (internal/hangout/domain):
--   type Hangout struct {
--       ID           uuid.UUID
--       OrganizerID  uuid.UUID   // FK to accounts.id
--       Title        string
--       Description  *string
--       Status       HangoutStatus // enum: "planned" | "ongoing" | "completed" | "cancelled"
--       ScheduledAt  *time.Time    // nullable: hangout may not have a fixed time yet
--       Metadata     map[string]any
--       Version      int
--       CreatedAt    time.Time
--       UpdatedAt    time.Time
--       DeletedAt    *time.Time
--   }
-- ============================================================
CREATE TABLE hangouts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id  UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    title         VARCHAR(150) NOT NULL,
    description   TEXT,
    status        VARCHAR(20) NOT NULL DEFAULT 'planned'
                  CHECK (status IN ('planned', 'ongoing', 'completed', 'cancelled')),
    scheduled_at  TIMESTAMPTZ,
    metadata      JSONB NOT NULL DEFAULT '{}',
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_hangouts_organizer_id ON hangouts(organizer_id);
CREATE INDEX idx_hangouts_status ON hangouts(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_hangouts_deleted_at ON hangouts(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER trg_hangouts_updated_at
    BEFORE UPDATE ON hangouts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- hangout_participants: one row per invited user per hangout.
-- This table covers BOTH the invite and the membership: a row is
-- created the moment a user is invited (invite_status = 'pending'),
-- and its status changes in place when the user responds. The
-- organizer also gets a row, role = 'organizer', status = 'accepted',
-- created at the same time as the hangout.
--
-- Go struct (internal/hangout/domain):
--   type Participant struct {
--       ID           uuid.UUID
--       HangoutID    uuid.UUID
--       UserID       uuid.UUID     // FK to accounts.id
--       Role         ParticipantRole // enum: "organizer" | "participant"
--       InviteStatus InviteStatus    // enum: "pending" | "accepted" | "rejected"
--       Reason       *string         // nullable: reason for a rejected invite
--       InvitedBy    uuid.UUID
--       InvitedAt    time.Time
--       RespondedAt  *time.Time
--       CreatedAt    time.Time
--       UpdatedAt    time.Time
--   }
-- No version/deleted_at — an invite response is a single, final
-- write; a removed participant is just a row we stop reading, not
-- something we soft-delete.
-- ============================================================
CREATE TABLE hangout_participants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hangout_id    UUID NOT NULL REFERENCES hangouts(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    role          VARCHAR(20) NOT NULL DEFAULT 'participant'
                  CHECK (role IN ('organizer', 'participant')),
    invite_status VARCHAR(20) NOT NULL DEFAULT 'pending'
                  CHECK (invite_status IN ('pending', 'accepted', 'rejected')),
    reason        TEXT,
    invited_by    UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    invited_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hangout_id, user_id)
);

CREATE INDEX idx_hangout_participants_hangout_id ON hangout_participants(hangout_id);
CREATE INDEX idx_hangout_participants_user_id ON hangout_participants(user_id);
CREATE INDEX idx_hangout_participants_invite_status ON hangout_participants(invite_status);


-- ============================================================
-- hangout_messages: group chat tied to one hangout. Only accepted
-- participants may post (enforced in the application layer).
--
-- Go struct (internal/hangout/domain):
--   type Message struct {
--       ID        uuid.UUID
--       HangoutID uuid.UUID
--       SenderID  uuid.UUID   // FK to accounts.id
--       Content   string
--       CreatedAt time.Time
--       DeletedAt *time.Time  // nil = not deleted; soft delete for moderation
--   }
-- ============================================================
CREATE TABLE hangout_messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hangout_id  UUID NOT NULL REFERENCES hangouts(id) ON DELETE CASCADE,
    sender_id   UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_hangout_messages_hangout_id_created_at
    ON hangout_messages(hangout_id, created_at);

-- +migrate Down

DROP TABLE IF EXISTS hangout_messages;
DROP TABLE IF EXISTS hangout_participants;
DROP TRIGGER IF EXISTS trg_hangouts_updated_at ON hangouts;
DROP TABLE IF EXISTS hangouts;
