-- +goose Up
CREATE TABLE users (
    id            BIGINT PRIMARY KEY,
    username      TEXT,
    first_name    TEXT NOT NULL,
    upi_vpa       TEXT,
    daily_summary BOOLEAN NOT NULL DEFAULT false,
    blocked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX users_username_idx ON users (lower(username));

CREATE TABLE chats (
    id         BIGINT PRIMARY KEY,
    kind       TEXT NOT NULL CHECK (kind IN ('private', 'group', 'supergroup')),
    title      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE processed_updates (
    update_id    BIGINT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX processed_updates_time_idx ON processed_updates (processed_at);

CREATE TABLE personal_expenses (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users (id),
    amount      BIGINT NOT NULL CHECK (amount > 0),
    category    TEXT NOT NULL,
    description TEXT NOT NULL,
    spent_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
CREATE INDEX personal_expenses_user_time_idx
    ON personal_expenses (user_id, spent_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE personal_expenses;
DROP TABLE processed_updates;
DROP TABLE chats;
DROP TABLE users;
