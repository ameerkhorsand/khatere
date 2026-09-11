-- +migrate Up

CREATE TABLE notifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type          VARCHAR(40) NOT NULL,
    recipient_id  UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    actor_id      UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    metadata      JSONB NOT NULL DEFAULT '{}',
    read_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_recipient ON notifications(recipient_id, created_at DESC);
CREATE INDEX idx_notifications_recipient_unread ON notifications(recipient_id) WHERE read_at IS NULL;

-- +migrate Down

DROP TABLE IF EXISTS notifications;
