-- name: EnqueueMessage :execrows
INSERT INTO outbox (chat_id, text, dedupe_key, next_try_at, created_at)
VALUES (sqlc.arg(chat_id), sqlc.arg(text), sqlc.arg(dedupe_key), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (dedupe_key) DO NOTHING;

-- name: ClaimMessage :one
UPDATE outbox
SET attempts = attempts + 1,
    next_try_at = sqlc.arg(lease_until)
WHERE id = (
    SELECT o.id FROM outbox o
    WHERE o.status = 'pending' AND o.next_try_at <= sqlc.arg(now)
    ORDER BY o.next_try_at, o.id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: MarkMessageSent :exec
UPDATE outbox
SET status = 'sent', last_error = NULL
WHERE id = $1;

-- name: MarkMessageFailed :exec
UPDATE outbox
SET status = 'failed', last_error = sqlc.arg(last_error)::TEXT
WHERE id = sqlc.arg(id);

-- name: RescheduleMessage :exec
UPDATE outbox
SET next_try_at = sqlc.arg(next_try_at),
    attempts = sqlc.arg(attempts),
    last_error = sqlc.arg(last_error)
WHERE id = sqlc.arg(id);

-- name: CountPendingMessages :one
SELECT count(*) FROM outbox
WHERE status = 'pending';

-- name: DeleteFinishedMessages :execrows
DELETE FROM outbox
WHERE status <> 'pending' AND created_at < sqlc.arg(before);
