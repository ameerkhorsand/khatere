-- +migrate Up

-- 0009 put an update trigger on hangout_meetup_pin_confirmations that
-- calls set_updated_at_and_bump_version(). That function always sets
-- NEW.version, but this table has no version column — so every
-- update to this table failed with a database error. This drops the
-- bad trigger. The corrected 0009 file no longer creates it, so a
-- fresh database will never have it in the first place; this
-- migration only matters for a database where the old 0009 already
-- ran.
DROP TRIGGER IF EXISTS trg_hangout_pin_confirmations_updated_at ON hangout_meetup_pin_confirmations;

-- +migrate Down

-- Deliberately does not recreate the trigger. It was broken (see
-- the Up section above); rolling back should not bring the bug back.
