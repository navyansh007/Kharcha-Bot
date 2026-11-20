// Package bot routes Telegram updates to command handlers and sends the replies.
package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/navyansh007/kharcha-bot/internal/money"
	"github.com/navyansh007/kharcha-bot/internal/parser"
	"github.com/navyansh007/kharcha-bot/internal/store"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

const (
	sendTimeout   = 10 * time.Second
	maxRetryDelay = 3 * time.Second
)

// Options configures a Bot. Now may be nil, in which case time.Now is used.
type Options struct {
	Pool        *pgxpool.Pool
	Client      telegram.Client
	Logger      *slog.Logger
	Location    *time.Location
	BotUsername string
	MaxExpense  money.Paise
	Now         func() time.Time
}

// Bot handles updates. It is safe for concurrent use.
type Bot struct {
	pool   *pgxpool.Pool
	client telegram.Client
	logger *slog.Logger
	loc    *time.Location
	parser parser.Parser
	now    func() time.Time
}

// New returns a Bot built from opts.
func New(opts Options) *Bot {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Bot{
		pool:   opts.Pool,
		client: opts.Client,
		logger: opts.Logger,
		loc:    opts.Location,
		parser: parser.Parser{MaxAmount: opts.MaxExpense, BotUsername: opts.BotUsername},
		now:    now,
	}
}

// Outcome says what happened to an update.
type Outcome string

// The possible outcomes of handling an update.
const (
	OutcomeProcessed Outcome = "processed"
	OutcomeDuplicate Outcome = "duplicate"
	OutcomeError     Outcome = "error"
)

// HandleUpdate applies the effects of an update exactly once and then sends the replies.
// It returns an error only if the effects were not committed and delivery should be retried.
func (b *Bot) HandleUpdate(ctx context.Context, update telegram.Update) error {
	start := time.Now()
	outcome := OutcomeProcessed
	var result response

	err := store.InTx(ctx, b.pool, func(q *db.Queries) error {
		rows, err := q.MarkUpdateProcessed(ctx, update.UpdateID)
		if err != nil {
			return fmt.Errorf("mark update: %w", err)
		}
		if rows == 0 {
			outcome = OutcomeDuplicate
			return nil
		}
		result, err = b.dispatch(ctx, q, update)
		return err
	})
	if err != nil {
		outcome = OutcomeError
	} else {
		b.deliver(context.WithoutCancel(ctx), result)
	}

	b.logger.Info("update handled",
		"update_id", update.UpdateID,
		"chat_id", result.chatID,
		"command", result.command,
		"outcome", outcome,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if err != nil {
		return fmt.Errorf("handle update %d: %w", update.UpdateID, err)
	}
	return nil
}

func (b *Bot) dispatch(ctx context.Context, q *db.Queries, update telegram.Update) (response, error) {
	switch {
	case update.Message != nil:
		return b.handleMessage(ctx, q, update.Message)
	case update.CallbackQuery != nil:
		return b.handleCallback(ctx, q, update.CallbackQuery)
	default:
		return response{}, nil
	}
}

func (b *Bot) deliver(ctx context.Context, r response) {
	if r.answer != nil {
		b.send(ctx, r.chatID, func(ctx context.Context) error {
			return b.client.AnswerCallbackQuery(ctx, *r.answer)
		})
	}
	for _, edit := range r.edits {
		b.send(ctx, edit.ChatID, func(ctx context.Context) error {
			return b.client.EditMessageText(ctx, edit)
		})
	}
	for _, message := range r.messages {
		b.send(ctx, message.ChatID, func(ctx context.Context) error {
			return b.client.SendMessage(ctx, message)
		})
	}
	for _, doc := range r.documents {
		b.send(ctx, doc.ChatID, func(ctx context.Context) error {
			return b.client.SendDocument(ctx, doc)
		})
	}
}

func (b *Bot) send(ctx context.Context, chatID int64, call func(context.Context) error) {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	err := call(ctx)
	var apiErr *telegram.APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 && apiErr.RetryAfter <= maxRetryDelay {
		select {
		case <-time.After(apiErr.RetryAfter):
			err = call(ctx)
		case <-ctx.Done():
		}
	}
	if err != nil {
		b.logger.Warn("send failed", "chat_id", chatID, "error", err)
	}
}
