-- +migrate Up

ALTER TABLE users
    ADD COLUMN handle CITEXT NOT NULL UNIQUE;

-- +migrate Down

ALTER TABLE users DROP COLUMN IF EXISTS handle;
