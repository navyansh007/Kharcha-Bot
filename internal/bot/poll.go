package bot

import (
	"context"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

const (
	pollTimeout  = 30 * time.Second
	pollBackoff  = 2 * time.Second
	maxAttempts  = 3
	handleBudget = 20 * time.Second
)

// Poll fetches updates with getUpdates and handles them one at a time until ctx is cancelled.
// An update that keeps failing is skipped so that it cannot block the ones after it.
func (b *Bot) Poll(ctx context.Context) {
	var offset int64
	attempts := 0

	for ctx.Err() == nil {
		updates, err := b.client.GetUpdates(ctx, offset, pollTimeout)
		if err != nil {
			if ctx.Err() == nil {
				b.logger.Warn("get updates failed", "error", err)
				sleep(ctx, pollBackoff)
			}
			continue
		}
		for _, update := range updates {
			if err := b.handlePolled(ctx, update); err != nil {
				attempts++
				b.logger.Error("update failed", "update_id", update.UpdateID, "attempt", attempts, "error", err)
				if attempts < maxAttempts {
					sleep(ctx, pollBackoff)
					break
				}
			}
			attempts = 0
			offset = update.UpdateID + 1
		}
	}
}

func (b *Bot) handlePolled(ctx context.Context, update telegram.Update) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), handleBudget)
	defer cancel()
	return b.HandleUpdate(ctx, update)
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}
