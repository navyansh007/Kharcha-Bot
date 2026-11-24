package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/export"
	"github.com/navyansh007/kharcha-bot/internal/format"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

func (b *Bot) setUPI(ctx context.Context, q *db.Queries, req request, res *response) error {
	err := q.SetUserVPA(ctx, db.SetUserVPAParams{ID: req.user.ID, Vpa: req.command.VPA})
	if err != nil {
		return fmt.Errorf("save upi id: %w", err)
	}
	res.reply(fmt.Sprintf("Saved your UPI ID as <code>%s</code>.", format.Escape(req.command.VPA)))
	return nil
}

func (b *Bot) summary(ctx context.Context, q *db.Queries, req request, res *response) error {
	if !req.private() {
		res.reply(privateOnlyText)
		return nil
	}
	enabled := req.command.Enabled
	err := q.SetDailySummary(ctx, db.SetDailySummaryParams{ID: req.user.ID, Enabled: enabled})
	if err != nil {
		return fmt.Errorf("save summary setting: %w", err)
	}
	if enabled {
		res.reply("Daily summary is on. It arrives at 9 pm on days you log something.")
		return nil
	}
	res.reply("Daily summary is off.")
	return nil
}

func (b *Bot) export(ctx context.Context, q *db.Queries, req request, res *response) error {
	if !req.private() {
		res.reply(privateOnlyText)
		return nil
	}
	content, found, err := export.CSV(ctx, q, req.user.ID, b.loc)
	if err != nil {
		return err
	}
	if !found {
		res.reply("There is nothing to export yet.")
		return nil
	}
	res.documents = append(res.documents, telegram.Document{
		ChatID:  req.chat.ID,
		Name:    "kharcha-" + req.now.In(b.loc).Format(time.DateOnly) + ".csv",
		Content: content,
	})
	return nil
}

const deleteWarning = "This erases your expenses, budgets and UPI ID. Group entries stay, " +
	"without your name, so that other members' balances remain correct.\n\n" +
	"Send <code>/deleteme confirm</code> to continue."

func (b *Bot) deleteMe(ctx context.Context, q *db.Queries, req request, res *response) error {
	if !req.private() {
		res.reply(privateOnlyText)
		return nil
	}
	if !req.command.Confirmed {
		res.reply(deleteWarning)
		return nil
	}
	steps := []func(context.Context, int64) error{
		q.DeleteUserExpenses,
		q.DeleteUserBudgets,
		q.DeleteUserBudgetAlerts,
		q.DeleteUserMessages,
		q.LeaveAllGroups,
		q.AnonymiseUser,
	}
	for _, step := range steps {
		if err := step(ctx, req.user.ID); err != nil {
			return fmt.Errorf("delete user data: %w", err)
		}
	}
	res.reply("Your personal data has been deleted.")
	return nil
}
