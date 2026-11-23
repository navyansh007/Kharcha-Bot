-- name: UpsertBudget :exec
INSERT INTO budgets (user_id, category, monthly)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, category) DO UPDATE
SET monthly = excluded.monthly;

-- name: GetBudget :one
SELECT monthly FROM budgets
WHERE user_id = $1 AND category = $2;

-- name: LockUser :exec
SELECT id FROM users
WHERE id = $1
FOR NO KEY UPDATE;

-- name: CategorySpend :one
SELECT COALESCE(SUM(amount), 0)::BIGINT AS total
FROM personal_expenses
WHERE user_id = sqlc.arg(user_id)
  AND category = sqlc.arg(category)
  AND deleted_at IS NULL
  AND spent_at >= sqlc.arg(from_time)
  AND spent_at < sqlc.arg(to_time);

-- name: InsertBudgetAlert :execrows
INSERT INTO budget_alerts (user_id, category, month, threshold)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;

-- name: DeleteBudgetAlerts :exec
DELETE FROM budget_alerts
WHERE user_id = $1 AND category = $2;
