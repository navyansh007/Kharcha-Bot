-- name: ExportPersonalExpenses :many
SELECT spent_at, category, description, amount
FROM personal_expenses
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY spent_at, id;

-- name: ExportGroupEntries :many
SELECT t.created_at, t.kind, t.description, c.title, e.amount
FROM group_entries e
JOIN group_txns t ON t.id = e.txn_id
JOIN chats c ON c.id = t.chat_id
WHERE e.user_id = $1
ORDER BY t.created_at, t.id;

-- name: DeleteUserExpenses :exec
DELETE FROM personal_expenses
WHERE user_id = $1;

-- name: DeleteUserBudgets :exec
DELETE FROM budgets
WHERE user_id = $1;

-- name: DeleteUserBudgetAlerts :exec
DELETE FROM budget_alerts
WHERE user_id = $1;

-- name: DeleteUserMessages :exec
DELETE FROM outbox
WHERE chat_id = $1;

-- name: LeaveAllGroups :exec
UPDATE group_members
SET left_at = now()
WHERE user_id = $1 AND left_at IS NULL;

-- name: AnonymiseUser :exec
UPDATE users
SET username = NULL,
    first_name = 'Deleted user',
    upi_vpa = NULL,
    daily_summary = false
WHERE id = $1;
