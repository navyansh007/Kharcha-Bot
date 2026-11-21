-- name: LockChat :exec
SELECT id FROM chats
WHERE id = $1
FOR NO KEY UPDATE;

-- name: InsertGroupTxn :one
INSERT INTO group_txns (chat_id, kind, created_by, description, total, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: InsertReversalTxn :one
INSERT INTO group_txns (chat_id, kind, created_by, description, total, reverses_id, created_at)
VALUES ($1, 'reversal', $2, $3, $4, $5, $6)
ON CONFLICT (reverses_id) DO NOTHING
RETURNING id;

-- name: InsertGroupEntry :exec
INSERT INTO group_entries (txn_id, user_id, amount)
VALUES ($1, $2, $3);

-- name: GetGroupTxn :one
SELECT * FROM group_txns
WHERE id = $1 AND chat_id = $2;

-- name: LastUndoableTxn :one
SELECT * FROM group_txns t
WHERE t.chat_id = $1
  AND t.created_by = $2
  AND t.kind <> 'reversal'
  AND NOT EXISTS (SELECT 1 FROM group_txns r WHERE r.reverses_id = t.id)
ORDER BY t.id DESC
LIMIT 1;

-- name: ListGroupEntries :many
SELECT user_id, amount FROM group_entries
WHERE txn_id = $1
ORDER BY user_id;

-- name: GroupBalances :many
SELECT u.id AS user_id, u.first_name, SUM(e.amount)::BIGINT AS balance
FROM group_entries e
JOIN group_txns t ON t.id = e.txn_id
JOIN users u ON u.id = e.user_id
WHERE t.chat_id = $1
GROUP BY u.id
HAVING SUM(e.amount) <> 0
ORDER BY u.id;
