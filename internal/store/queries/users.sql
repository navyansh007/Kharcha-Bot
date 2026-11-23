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
    title = excluded.title,
    removed_at = NULL;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: SetUserVPA :exec
UPDATE users
SET upi_vpa = sqlc.arg(vpa)::TEXT
WHERE id = sqlc.arg(id);

-- name: MarkUserBlocked :exec
UPDATE users
SET blocked_at = now()
WHERE id = $1;

-- name: MarkChatRemoved :exec
UPDATE chats
SET removed_at = now()
WHERE id = $1;
