-- name: SetDailySummary :exec
UPDATE users
SET daily_summary = sqlc.arg(enabled)
WHERE id = sqlc.arg(id);

-- name: SummaryRecipients :many
SELECT u.id FROM users u
WHERE u.daily_summary
  AND u.blocked_at IS NULL
  AND EXISTS (
      SELECT 1 FROM personal_expenses e
      WHERE e.user_id = u.id
        AND e.deleted_at IS NULL
        AND e.spent_at >= sqlc.arg(from_time)
        AND e.spent_at < sqlc.arg(to_time)
  )
ORDER BY u.id;

-- name: GroupsWithOpenBalances :many
SELECT DISTINCT b.chat_id
FROM (
    SELECT t.chat_id
    FROM group_txns t
    JOIN group_entries e ON e.txn_id = t.id
    GROUP BY t.chat_id, e.user_id
    HAVING SUM(e.amount) <> 0
) b
JOIN chats c ON c.id = b.chat_id
WHERE c.removed_at IS NULL
ORDER BY b.chat_id;

-- name: DeleteProcessedUpdates :execrows
DELETE FROM processed_updates
WHERE processed_at < sqlc.arg(before);
