// Package outbox queues notifications and delivers them within Telegram's rate limits.
package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

const (
	maxAttempts    = 5
	lease          = 2 * time.Minute
	sendTimeout    = 10 * time.Second
	globalPerSec   = 25
	privateSpacing = time.Second
	groupSpacing   = 3 * time.Second
)

// Sender delivers one message to a chat.
type Sender interface {
	Send(ctx context.Context, chatID int64, text string) error
}

// SendError tells the outbox how to treat a failed delivery. Errors of other
// types are retried with backoff.
type SendError struct {
	Err        error
	RetryAfter time.Duration
	Permanent  bool
	Blocked    bool
}

func (e *SendError) Error() string {
	return e.Err.Error()
}

func (e *SendError) Unwrap() error {
	return e.Err
}

// Enqueue adds a message for later delivery. A message whose dedupeKey was
// already queued is ignored, and Enqueue reports false.
func Enqueue(
	ctx context.Context,
	q *db.Queries,
	chatID int64,
	text, dedupeKey string,
	now time.Time,
) (bool, error) {
	rows, err := q.EnqueueMessage(ctx, db.EnqueueMessageParams{
		ChatID:    chatID,
		Text:      text,
		DedupeKey: &dedupeKey,
		Now:       now,
	})
	if err != nil {
		return false, fmt.Errorf("enqueue message: %w", err)
	}
	return rows > 0, nil
}

// Stats counts the outcome of one drain.
type Stats struct {
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
	Delayed int `json:"delayed"`
}

// Drainer sends pending messages.
type Drainer struct {
	pool   *pgxpool.Pool
	sender Sender
	logger *slog.Logger
	global *rate.Limiter
}

// NewDrainer returns a Drainer that delivers through sender.
func NewDrainer(pool *pgxpool.Pool, sender Sender, logger *slog.Logger) *Drainer {
	return &Drainer{
		pool:   pool,
		sender: sender,
		logger: logger,
		global: rate.NewLimiter(globalPerSec, 1),
	}
}

// Drain sends due messages until none are left or budget has elapsed. Delivery
// is at least once: a crash between sending and recording repeats the message.
func (d *Drainer) Drain(ctx context.Context, budget time.Duration) (Stats, error) {
	var stats Stats
	deadline := time.Now().Add(budget)
	chats := map[int64]*rate.Limiter{}
	q := db.New(d.pool)

	for ctx.Err() == nil && time.Now().Before(deadline) {
		now := time.Now()
		msg, err := q.ClaimMessage(ctx, db.ClaimMessageParams{LeaseUntil: now.Add(lease), Now: now})
		if errors.Is(err, pgx.ErrNoRows) {
			return stats, nil
		}
		if err != nil {
			return stats, fmt.Errorf("claim message: %w", err)
		}

		limiter, ok := chats[msg.ChatID]
		if !ok {
			limiter = newChatLimiter(msg.ChatID)
			chats[msg.ChatID] = limiter
		}
		delay := max(limiter.Reserve().Delay(), d.global.Reserve().Delay())
		if time.Now().Add(delay).After(deadline) {
			stats.Delayed++
			err = reschedule(ctx, q, msg, time.Now().Add(delay), msg.Attempts-1, msg.LastError)
		} else {
			time.Sleep(delay)
			err = d.deliver(ctx, q, msg, &stats)
		}
		if err != nil {
			return stats, err
		}
	}
	return stats, nil
}

func newChatLimiter(chatID int64) *rate.Limiter {
	spacing := privateSpacing
	if chatID < 0 {
		spacing = groupSpacing
	}
	return rate.NewLimiter(rate.Every(spacing), 1)
}

func (d *Drainer) deliver(ctx context.Context, q *db.Queries, msg db.Outbox, stats *Stats) error {
	ctx = context.WithoutCancel(ctx)
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	sendErr := d.sender.Send(sendCtx, msg.ChatID, msg.Text)
	if sendErr == nil {
		stats.Sent++
		if err := q.MarkMessageSent(ctx, msg.ID); err != nil {
			return fmt.Errorf("mark sent: %w", err)
		}
		return nil
	}
	d.logger.Warn("outbox send failed", "outbox_id", msg.ID, "chat_id", msg.ChatID, "error", sendErr)

	var failure *SendError
	if !errors.As(sendErr, &failure) {
		failure = &SendError{Err: sendErr}
	}
	reason := sendErr.Error()

	switch {
	case failure.RetryAfter > 0:
		stats.Delayed++
		return reschedule(ctx, q, msg, time.Now().Add(failure.RetryAfter), msg.Attempts-1, &reason)
	case failure.Blocked || failure.Permanent || msg.Attempts >= maxAttempts:
		stats.Failed++
		if failure.Blocked {
			if err := markUnreachable(ctx, q, msg.ChatID); err != nil {
				return err
			}
		}
		err := q.MarkMessageFailed(ctx, db.MarkMessageFailedParams{ID: msg.ID, LastError: reason})
		if err != nil {
			return fmt.Errorf("mark failed: %w", err)
		}
		return nil
	default:
		stats.Delayed++
		return reschedule(ctx, q, msg, time.Now().Add(backoff(msg.Attempts)), msg.Attempts, &reason)
	}
}

func backoff(attempts int32) time.Duration {
	return time.Minute << (attempts - 1)
}

func markUnreachable(ctx context.Context, q *db.Queries, chatID int64) error {
	mark := q.MarkUserBlocked
	if chatID < 0 {
		mark = q.MarkChatRemoved
	}
	if err := mark(ctx, chatID); err != nil {
		return fmt.Errorf("mark chat unreachable: %w", err)
	}
	return nil
}

func reschedule(
	ctx context.Context,
	q *db.Queries,
	msg db.Outbox,
	at time.Time,
	attempts int32,
	reason *string,
) error {
	err := q.RescheduleMessage(ctx, db.RescheduleMessageParams{
		ID:        msg.ID,
		NextTryAt: at,
		Attempts:  attempts,
		LastError: reason,
	})
	if err != nil {
		return fmt.Errorf("reschedule message: %w", err)
	}
	return nil
}
