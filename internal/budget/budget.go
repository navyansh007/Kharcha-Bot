// Package budget tracks monthly category budgets and their warnings.
package budget

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/navyansh007/kharcha-bot/internal/ledger"
	"github.com/navyansh007/kharcha-bot/internal/money"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

var thresholds = []int16{80, 100}

// Warning reports that spending has crossed a threshold of a budget.
type Warning struct {
	Category  string
	Threshold int16
	Spent     money.Paise
	Monthly   money.Paise
}

// Set creates or replaces a monthly budget. Earlier warnings are forgotten so
// that the new limit can warn again.
func Set(ctx context.Context, q *db.Queries, userID int64, category string, monthly money.Paise) error {
	err := q.UpsertBudget(ctx, db.UpsertBudgetParams{UserID: userID, Category: category, Monthly: monthly})
	if err != nil {
		return fmt.Errorf("save budget: %w", err)
	}
	err = q.DeleteBudgetAlerts(ctx, db.DeleteBudgetAlertsParams{UserID: userID, Category: category})
	if err != nil {
		return fmt.Errorf("reset budget alerts: %w", err)
	}
	return nil
}

// Check returns a warning the first time in a month that spending in a category
// reaches a threshold. It returns nil if there is nothing new to report.
func Check(
	ctx context.Context,
	q *db.Queries,
	userID int64,
	category string,
	now time.Time,
	loc *time.Location,
) (*Warning, error) {
	monthly, err := q.GetBudget(ctx, db.GetBudgetParams{UserID: userID, Category: category})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get budget: %w", err)
	}

	if err := q.LockUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("lock user: %w", err)
	}
	period := ledger.Month(now, loc)
	total, err := q.CategorySpend(ctx, db.CategorySpendParams{
		UserID:   userID,
		Category: category,
		FromTime: period.From,
		ToTime:   period.To,
	})
	if err != nil {
		return nil, fmt.Errorf("month spend: %w", err)
	}
	spent := money.Paise(total)
	year, month, _ := period.From.Date()

	var warning *Warning
	for _, threshold := range thresholds {
		if !crossed(spent, monthly, threshold) {
			continue
		}
		rows, err := q.InsertBudgetAlert(ctx, db.InsertBudgetAlertParams{
			UserID:    userID,
			Category:  category,
			Month:     time.Date(year, month, 1, 0, 0, 0, 0, time.UTC),
			Threshold: threshold,
		})
		if err != nil {
			return nil, fmt.Errorf("save budget alert: %w", err)
		}
		if rows > 0 {
			warning = &Warning{Category: category, Threshold: threshold, Spent: spent, Monthly: monthly}
		}
	}
	return warning, nil
}

func crossed(spent, monthly money.Paise, threshold int16) bool {
	return spent*100 >= monthly*money.Paise(threshold)
}
