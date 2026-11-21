-- +goose Up
CREATE TABLE group_members (
    chat_id   BIGINT NOT NULL REFERENCES chats (id),
    user_id   BIGINT NOT NULL REFERENCES users (id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at   TIMESTAMPTZ,
    PRIMARY KEY (chat_id, user_id)
);

CREATE TABLE group_txns (
    id          BIGSERIAL PRIMARY KEY,
    chat_id     BIGINT NOT NULL REFERENCES chats (id),
    kind        TEXT NOT NULL CHECK (kind IN ('expense', 'settlement', 'reversal')),
    created_by  BIGINT NOT NULL REFERENCES users (id),
    description TEXT NOT NULL,
    total       BIGINT NOT NULL CHECK (total > 0),
    reverses_id BIGINT UNIQUE REFERENCES group_txns (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX group_txns_chat_idx ON group_txns (chat_id, created_at DESC);

CREATE TABLE group_entries (
    txn_id  BIGINT NOT NULL REFERENCES group_txns (id),
    user_id BIGINT NOT NULL REFERENCES users (id),
    amount  BIGINT NOT NULL,
    PRIMARY KEY (txn_id, user_id)
);
CREATE INDEX group_entries_user_idx ON group_entries (user_id);

-- +goose StatementBegin
CREATE FUNCTION check_group_txn_balanced() RETURNS trigger AS $$
BEGIN
    IF (SELECT SUM(amount) FROM group_entries WHERE txn_id = NEW.txn_id) <> 0 THEN
        RAISE EXCEPTION 'entries of group txn % do not sum to zero', NEW.txn_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER group_entries_balanced
    AFTER INSERT OR UPDATE ON group_entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_group_txn_balanced();

-- +goose Down
DROP TABLE group_entries;
DROP FUNCTION check_group_txn_balanced();
DROP TABLE group_txns;
DROP TABLE group_members;
