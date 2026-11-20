package bot

import (
	"context"
	"errors"
	"fmt"

	"github.com/navyansh007/kharcha-bot/internal/category"
	"github.com/navyansh007/kharcha-bot/internal/format"
	"github.com/navyansh007/kharcha-bot/internal/ledger"
	"github.com/navyansh007/kharcha-bot/internal/parser"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

const privateOnlyText = "That works only in a private chat with me."

const undoExpensePrefix = "pundo"

func (b *Bot) logExpense(ctx context.Context, q *db.Queries, req request, res *response) error {
	name := req.command.Tag
	if name == "" {
		name = category.Detect(req.command.Description)
	}
	id, err := ledger.LogExpense(ctx, q, ledger.Expense{
		UserID:      req.user.ID,
		Amount:      req.command.Amount,
		Category:    name,
		Description: req.command.Description,
	}, req.now)
	if err != nil {
		return err
	}

	text := fmt.Sprintf("✅ %s %s logged under %s.",
		req.command.Amount, format.Escape(req.command.Description), format.Escape(name))
	res.reply(text, row(button("Undo", fmt.Sprintf("%s:%d", undoExpensePrefix, id))))
	return nil
}

func (b *Bot) totals(ctx context.Context, q *db.Queries, req request, res *response) error {
	if !req.private() {
		res.reply(privateOnlyText)
		return nil
	}
	title, period := "Today", ledger.Day(req.now, b.loc)
	switch req.command.Kind {
	case parser.KindWeek:
		title, period = "This week", ledger.Week(req.now, b.loc)
	case parser.KindMonth:
		title, period = "This month", ledger.Month(req.now, b.loc)
	}

	totals, err := ledger.Totals(ctx, q, req.user.ID, period)
	if err != nil {
		return err
	}
	res.reply(format.Totals(title, totals))
	return nil
}

func (b *Bot) undoLast(ctx context.Context, q *db.Queries, req request, res *response) error {
	if !req.private() {
		res.reply(privateOnlyText)
		return nil
	}
	expense, err := ledger.UndoLastExpense(ctx, q, req.user.ID, req.now)
	if text, ok := undoFailure(err); ok {
		res.reply(text)
		return nil
	}
	if err != nil {
		return err
	}
	res.reply(fmt.Sprintf("↩️ Removed %s %s.", expense.Amount, format.Escape(expense.Description)))
	return nil
}

func undoFailure(err error) (string, bool) {
	switch {
	case errors.Is(err, ledger.ErrNotFound):
		return "There is nothing to undo.", true
	case errors.Is(err, ledger.ErrNotOwner):
		return "Only the person who added it can undo it.", true
	case errors.Is(err, ledger.ErrTooLate):
		return "Too late to undo. That is only possible for 10 minutes.", true
	case errors.Is(err, ledger.ErrAlreadyUndone):
		return "Already undone.", true
	default:
		return "", false
	}
}
