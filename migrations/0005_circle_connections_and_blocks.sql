-- +migrate Up

-- ============================================================
-- circle_connections: symmetric friendship state between two users.
-- A single row represents the relationship between requester and addressee,
-- regardless of who initiated it. A canonical-order check constraint
-- guarantees A<->B never gets two rows (one as requester, one as addressee).
-- ============================================================
CREATE TABLE circle_connections (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id  UUID NOT NULL REFERENCES users(account_id) ON DELETE CASCADE,
    addressee_id  UUID NOT NULL REFERENCES users(account_id) ON DELETE CASCADE,
    status        VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'accepted')),
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_no_self_connection CHECK (requester_id <> addressee_id),
    -- Canonical pair: the smaller UUID always goes in requester_id_ordered,
    -- enforced via a generated column, so (A,B) and (B,A) can never both exist.
    canonical_low  UUID GENERATED ALWAYS AS (LEAST(requester_id, addressee_id)) STORED,
    canonical_high UUID GENERATED ALWAYS AS (GREATEST(requester_id, addressee_id)) STORED,
    UNIQUE (canonical_low, canonical_high)
);

CREATE INDEX idx_circle_connections_requester ON circle_connections(requester_id) WHERE status = 'pending';
CREATE INDEX idx_circle_connections_addressee ON circle_connections(addressee_id) WHERE status = 'pending';
CREATE INDEX idx_circle_connections_accepted_low ON circle_connections(canonical_low) WHERE status = 'accepted';
CREATE INDEX idx_circle_connections_accepted_high ON circle_connections(canonical_high) WHERE status = 'accepted';

CREATE TRIGGER trg_circle_connections_updated_at
    BEFORE UPDATE ON circle_connections
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at_and_bump_version();

-- ============================================================
-- circle_blocks: directional. Blocking is one-sided and independent
-- of any connection row — it must survive/override connection state,
-- so it is not merged into circle_connections.
-- ============================================================
CREATE TABLE circle_blocks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_id UUID NOT NULL REFERENCES users(account_id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(account_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_no_self_block CHECK (blocker_id <> blocked_id),
    UNIQUE (blocker_id, blocked_id)
);

CREATE INDEX idx_circle_blocks_blocker ON circle_blocks(blocker_id);
CREATE INDEX idx_circle_blocks_blocked ON circle_blocks(blocked_id);

-- +migrate Down

DROP TABLE IF EXISTS circle_blocks;
DROP TRIGGER IF EXISTS trg_circle_connections_updated_at ON circle_connections;
DROP TABLE IF EXISTS circle_connections;
