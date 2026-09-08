-- +migrate Up

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;     -- for case-insensitive email

-- Reusable trigger function: bumps updated_at and version on every row update.
CREATE OR REPLACE FUNCTION set_updated_at_and_bump_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- accounts: shared identity/auth. One row per login credential.
--
-- Go struct (internal/auth/domain):
--   type Account struct {
--       ID           uuid.UUID
--       Email        string
--       PasswordHash string
--       AccountType  AccountType   // enum: "user" | "host" | "moderator"
--       Metadata     map[string]any
--       Version      int
--       CreatedAt    time.Time
--       UpdatedAt    time.Time
--       DeletedAt    *time.Time    // nil = not deleted
--   }
-- ============================================================
CREATE TABLE accounts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    account_type  VARCHAR(20) NOT NULL CHECK (account_type IN ('user', 'host', 'moderator')),
    metadata      JSONB NOT NULL DEFAULT '{}',
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_accounts_account_type ON accounts(account_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_accounts_deleted_at ON accounts(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER trg_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- refresh_tokens: many per account.
--
-- Go struct (internal/auth/domain):
--   type RefreshToken struct {
--       ID        uuid.UUID
--       AccountID uuid.UUID
--       TokenHash string
--       ExpiresAt time.Time
--       Revoked   bool
--       CreatedAt time.Time
--   }
-- No version/deleted_at — a token's lifecycle is fully described
-- by expires_at and revoked, so soft delete/locking add nothing here.
-- ============================================================
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_account_id ON refresh_tokens(account_id);

-- ============================================================
-- users: regular user profile. 1:1 with an account.
--
-- Go struct (internal/user/domain):
--   type User struct {
--       AccountID   uuid.UUID   // shared PK/FK with accounts.id
--       DisplayName string
--       Bio         *string     // nullable
--       Metadata    map[string]any
--       Version     int
--       CreatedAt   time.Time
--       UpdatedAt   time.Time
--       DeletedAt   *time.Time
--   }
-- ============================================================
CREATE TABLE users (
    account_id   UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    display_name VARCHAR(100) NOT NULL,
    bio          TEXT,
    metadata     JSONB NOT NULL DEFAULT '{}',
    version      INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- hosts: host profile. 1:1 with an account.
--
-- Go struct (internal/host/domain):
--   type Host struct {
--       AccountID    uuid.UUID
--       BusinessName string
--       LocationInfo *string
--       Metadata     map[string]any
--       Version      int
--       CreatedAt    time.Time
--       UpdatedAt    time.Time
--       DeletedAt    *time.Time
--   }
-- ============================================================
CREATE TABLE hosts (
    account_id    UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    business_name VARCHAR(150) NOT NULL,
    location_info TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}',
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_hosts_deleted_at ON hosts(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER trg_hosts_updated_at
    BEFORE UPDATE ON hosts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- moderators: moderator profile. 1:1 with an account.
--
-- Go struct (internal/moderator/domain):
--   type Moderator struct {
--       AccountID uuid.UUID
--       Metadata  map[string]any
--       Version   int
--       CreatedAt time.Time
--       UpdatedAt time.Time
--       DeletedAt *time.Time
--   }
-- ============================================================
CREATE TABLE moderators (
    account_id UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    metadata   JSONB NOT NULL DEFAULT '{}',
    version    INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_moderators_deleted_at ON moderators(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER trg_moderators_updated_at
    BEFORE UPDATE ON moderators
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- user_interests: a user's stated interests. Many rows per user.
--
-- Go struct (internal/user/domain):
--   type UserInterest struct {
--       ID        uuid.UUID
--       UserID    uuid.UUID   // FK to users.account_id
--       Interest  string
--       CreatedAt time.Time
--   }
-- No version/deleted_at — a tag list; removed rows are just deleted.
-- ============================================================
CREATE TABLE user_interests (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(account_id) ON DELETE CASCADE,
    interest   VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, interest)
);

CREATE INDEX idx_user_interests_user_id ON user_interests(user_id);

-- +migrate Down
DROP TABLE IF EXISTS user_interests;
DROP TRIGGER IF EXISTS trg_moderators_updated_at ON moderators;
DROP TABLE IF EXISTS moderators;
DROP TRIGGER IF EXISTS trg_hosts_updated_at ON hosts;
DROP TABLE IF EXISTS hosts;
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS refresh_tokens;
DROP TRIGGER IF EXISTS trg_accounts_updated_at ON accounts;
DROP TABLE IF EXISTS accounts;
DROP FUNCTION IF EXISTS set_updated_at_and_bump_version();
