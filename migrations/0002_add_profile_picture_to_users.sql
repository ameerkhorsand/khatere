-- +migrate Up

ALTER TABLE users
    ADD COLUMN profile_picture_key TEXT;

-- +migrate Down

ALTER TABLE users
    DROP COLUMN IF EXISTS profile_picture_key;
