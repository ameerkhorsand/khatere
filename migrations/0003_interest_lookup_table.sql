-- +migrate Up

-- Lookup table: a fixed, curated list of interests. New interests are just
-- inserted here — no schema migration needed to add one.
CREATE TABLE interests (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       VARCHAR(50) NOT NULL UNIQUE,
    label      VARCHAR(100) NOT NULL,
    category   VARCHAR(50),
    active     BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_interests_category ON interests(category) WHERE active = true;

-- Seed a starter list. Add more with plain INSERTs later — no migration needed.
INSERT INTO interests (slug, label, category, sort_order) VALUES
    ('hiking',      'Hiking',       'outdoor', 1),
    ('board_games', 'Board Games',  'indoor',  2),
    ('cooking',     'Cooking',      'indoor',  3),
    ('cycling',     'Cycling',      'outdoor', 4),
    ('movies',      'Movies',       'indoor',  5),
    ('live_music',  'Live Music',   'social',  6);

-- Rebuild user_interests to reference the lookup table instead of free text.
DROP TABLE IF EXISTS user_interests;

CREATE TABLE user_interests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(account_id) ON DELETE CASCADE,
    interest_id UUID NOT NULL REFERENCES interests(id) ON DELETE RESTRICT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, interest_id)
);

CREATE INDEX idx_user_interests_user_id ON user_interests(user_id);
CREATE INDEX idx_user_interests_interest_id ON user_interests(interest_id);

-- +migrate Down
DROP TABLE IF EXISTS user_interests;
DROP TABLE IF EXISTS interests;

CREATE TABLE user_interests (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(account_id) ON DELETE CASCADE,
    interest   VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, interest)
);

CREATE INDEX idx_user_interests_user_id ON user_interests(user_id);
