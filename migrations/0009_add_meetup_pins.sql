-- +migrate Up

-- ============================================================
-- hangout_meetup_pins: the proposed time and place for one
-- hangout. Only one active pin exists per hangout. A new
-- proposal replaces the old data in place and bumps the version.
--
-- Go struct (internal/hangout/domain):
--   type MeetupPin struct {
--       ID          uuid.UUID
--       HangoutID   uuid.UUID
--       PlaceName   string
--       Address     *string
--       Latitude    float64
--       Longitude   float64
--       ScheduledAt *time.Time
--       ProposedBy  uuid.UUID
--       Version     int
--       CreatedAt   time.Time
--       UpdatedAt   time.Time
--   }
-- ============================================================
CREATE TABLE hangout_meetup_pins (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hangout_id    UUID NOT NULL REFERENCES hangouts(id) ON DELETE CASCADE,
    place_name    VARCHAR(200) NOT NULL,
    address       TEXT,
    latitude      DOUBLE PRECISION NOT NULL,
    longitude     DOUBLE PRECISION NOT NULL,
    scheduled_at  TIMESTAMPTZ,
    proposed_by   UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hangout_id)
);

CREATE INDEX idx_hangout_meetup_pins_hangout_id ON hangout_meetup_pins(hangout_id);

CREATE TRIGGER trg_hangout_meetup_pins_updated_at
    BEFORE UPDATE ON hangout_meetup_pins
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- hangout_meetup_pin_confirmations: one row per active
-- participant per pin. A change to the pin resets every row
-- back to 'pending'. There is no deadline field by design.
--
-- Go struct (internal/hangout/domain):
--   type PinConfirmation struct {
--       ID          uuid.UUID
--       PinID       uuid.UUID
--       UserID      uuid.UUID
--       Status      PinConfirmationStatus // "pending" | "confirmed" | "declined"
--       ConfirmedAt *time.Time
--       CreatedAt   time.Time
--       UpdatedAt   time.Time
--   }
-- ============================================================
CREATE TABLE hangout_meetup_pin_confirmations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pin_id        UUID NOT NULL REFERENCES hangout_meetup_pins(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'confirmed', 'declined')),
    confirmed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pin_id, user_id)
);

CREATE INDEX idx_hangout_pin_confirmations_pin_id ON hangout_meetup_pin_confirmations(pin_id);
CREATE INDEX idx_hangout_pin_confirmations_user_id ON hangout_meetup_pin_confirmations(user_id);

-- No update trigger here, unlike the other hangout tables. This
-- table has no version column, so set_updated_at_and_bump_version()
-- would fail on every update. UpdateConfirmation (in the Go
-- repository) sets updated_at by hand instead.

-- +migrate Down

DROP TABLE IF EXISTS hangout_meetup_pin_confirmations;
DROP TABLE IF EXISTS hangout_meetup_pins;
