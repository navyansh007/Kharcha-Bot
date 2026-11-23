package bot

import (
	"context"
	"fmt"

	"github.com/navyansh007/kharcha-bot/internal/format"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
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
