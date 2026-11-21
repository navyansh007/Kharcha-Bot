-- name: UpsertMember :exec
INSERT INTO group_members (chat_id, user_id)
VALUES ($1, $2)
ON CONFLICT (chat_id, user_id) DO UPDATE
SET left_at = NULL;

-- name: MarkMemberLeft :exec
UPDATE group_members
SET left_at = now()
WHERE chat_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: ListMembers :many
SELECT u.id, u.first_name, u.username
FROM group_members m
JOIN users u ON u.id = m.user_id
WHERE m.chat_id = $1 AND m.left_at IS NULL
ORDER BY u.id;

-- name: GetMember :one
SELECT u.id, u.first_name, u.username
FROM group_members m
JOIN users u ON u.id = m.user_id
WHERE m.chat_id = $1 AND m.user_id = $2 AND m.left_at IS NULL;

-- name: FindMemberByUsername :one
SELECT u.id, u.first_name, u.username
FROM group_members m
JOIN users u ON u.id = m.user_id
WHERE m.chat_id = sqlc.arg(chat_id)
  AND m.left_at IS NULL
  AND lower(u.username) = lower(sqlc.arg(username)::TEXT)
ORDER BY m.joined_at DESC
LIMIT 1;
