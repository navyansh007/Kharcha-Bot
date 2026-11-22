-- name: UpsertUser :exec
INSERT INTO users (id, username, first_name)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET username = excluded.username,
    first_name = excluded.first_name,
    blocked_at = NULL;

-- name: UpsertChat :exec
INSERT INTO chats (id, kind, title)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET kind = excluded.kind,
    title = excluded.title;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;
