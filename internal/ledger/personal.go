package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/navyansh007/kharcha-bot/internal/money"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

// Expense is a personal expense to be logged.
type Expense struct {
	UserID      int64
	Amount      money.Paise
	Category    string
	Description string
}

// CategoryTotal is the amount spent in one category.
type CategoryTotal struct {
	Category string
	Total    money.Paise
}

// LogExpense records a personal expense and returns its ID.
func LogExpense(ctx context.Context, q *db.Queries, e Expense, now time.Time) (int64, error) {
	id, err := q.InsertPersonalExpense(ctx, db.InsertPersonalExpenseParams{
		UserID:      e.UserID,
		Amount:      e.Amount,
		Category:    e.Category,
		Description: e.Description,
		SpentAt:     now,
	})
	if err != nil {
		return 0, fmt.Errorf("insert expense: %w", err)
	}
	return id, nil
}

// UndoExpense deletes the user's expense with the given ID if it is recent enough.
func UndoExpense(ctx context.Context, q *db.Queries, userID, id int64, now time.Time) (db.PersonalExpense, error) {
	expense, err := q.GetPersonalExpense(ctx, db.GetPersonalExpenseParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.PersonalExpense{}, ErrNotFound
	}
	if err != nil {
		return db.PersonalExpense{}, fmt.Errorf("get expense: %w", err)
	}
	return expense, deleteExpense(ctx, q, expense, now)
}

// UndoLastExpense deletes the user's most recent expense if it is recent enough.
func UndoLastExpense(ctx context.Context, q *db.Queries, userID int64, now time.Time) (db.PersonalExpense, error) {
	expense, err := q.LastPersonalExpense(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.PersonalExpense{}, ErrNotFound
	}
	if err != nil {
		return db.PersonalExpense{}, fmt.Errorf("get last expense: %w", err)
	}
	return expense, deleteExpense(ctx, q, expense, now)
}

func deleteExpense(ctx context.Context, q *db.Queries, e db.PersonalExpense, now time.Time) error {
	if e.DeletedAt != nil {
		return ErrAlreadyUndone
	}
	if now.Sub(e.SpentAt) > UndoWindow {
		return ErrTooLate
	}
	rows, err := q.DeletePersonalExpense(ctx, db.DeletePersonalExpenseParams{
		ID:        e.ID,
		UserID:    e.UserID,
		DeletedAt: now,
	})
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if rows == 0 {
		return ErrAlreadyUndone
	}
	return nil
}

// Totals returns the user's spending in the period, largest category first.
func Totals(ctx context.Context, q *db.Queries, userID int64, p Period) ([]CategoryTotal, error) {
	rows, err := q.PersonalTotalsByCategory(ctx, db.PersonalTotalsByCategoryParams{
		UserID:   userID,
		FromTime: p.From,
		ToTime:   p.To,
	})
	if err != nil {
		return nil, fmt.Errorf("category totals: %w", err)
	}
	totals := make([]CategoryTotal, len(rows))
	for i, row := range rows {
		totals[i] = CategoryTotal{Category: row.Category, Total: money.Paise(row.Total)}
	}
	return totals, nil
}
