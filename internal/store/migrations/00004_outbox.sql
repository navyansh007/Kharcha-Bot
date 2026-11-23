-- +goose Up
CREATE TABLE outbox (
    id          BIGSERIAL PRIMARY KEY,
    chat_id     BIGINT NOT NULL,
    text        TEXT NOT NULL,
    dedupe_key  TEXT UNIQUE,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    attempts    INT NOT NULL DEFAULT 0,
    next_try_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX outbox_pending_idx ON outbox (next_try_at) WHERE status = 'pending';

ALTER TABLE chats ADD COLUMN removed_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE chats DROP COLUMN removed_at;
DROP TABLE outbox;
