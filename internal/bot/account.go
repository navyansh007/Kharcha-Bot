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
