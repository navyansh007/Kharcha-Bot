-- name: InsertPersonalExpense :one
INSERT INTO personal_expenses (user_id, amount, category, description, spent_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetPersonalExpense :one
SELECT * FROM personal_expenses
WHERE id = $1 AND user_id = $2;

-- name: LastPersonalExpense :one
SELECT * FROM personal_expenses
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY spent_at DESC, id DESC
LIMIT 1;

-- name: DeletePersonalExpense :execrows
UPDATE personal_expenses
SET deleted_at = sqlc.arg(deleted_at)::TIMESTAMPTZ
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: PersonalTotalsByCategory :many
SELECT category, SUM(amount)::BIGINT AS total
FROM personal_expenses
WHERE user_id = sqlc.arg(user_id)
  AND deleted_at IS NULL
  AND spent_at >= sqlc.arg(from_time)
  AND spent_at < sqlc.arg(to_time)
GROUP BY category
ORDER BY total DESC, category;
