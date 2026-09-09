-- +migrate Up

-- When a block is inserted, delete any existing connection (pending or
-- accepted) between the two users, in either direction. This guarantees
-- the rule holds at the database level, even if application code forgets
-- to call the cleanup itself.
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION delete_connection_on_block()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM circle_connections
    WHERE (requester_id = NEW.blocker_id AND addressee_id = NEW.blocked_id)
       OR (requester_id = NEW.blocked_id AND addressee_id = NEW.blocker_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TRIGGER trg_delete_connection_on_block
    AFTER INSERT ON circle_blocks
    FOR EACH ROW
    EXECUTE FUNCTION delete_connection_on_block();

-- +migrate Down

DROP TRIGGER IF EXISTS trg_delete_connection_on_block ON circle_blocks;
DROP FUNCTION IF EXISTS delete_connection_on_block();
