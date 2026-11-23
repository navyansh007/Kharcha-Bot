// Package cron runs the scheduled jobs. Every job is safe to run twice.
package cron

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/navyansh007/kharcha-bot/internal/format"
	"github.com/navyansh007/kharcha-bot/internal/ledger"
	"github.com/navyansh007/kharcha-bot/internal/outbox"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

// The names of the jobs.
const (
	DailySummary   = "daily-summary"
	WeeklyReminder = "weekly-reminder"
	DrainOutbox    = "drain-outbox"
	Cleanup        = "cleanup"
)

const (
	drainBudget     = 25 * time.Second
	updateRetention = 7 * 24 * time.Hour
	outboxRetention = 30 * 24 * time.Hour
)

// ErrUnknownJob is returned for a job name that does not exist.
var ErrUnknownJob = errors.New("unknown job")

// Result counts what a job did.
type Result struct {
	Enqueued int          `json:"enqueued"`
	Deleted  int64        `json:"deleted"`
	Outbox   outbox.Stats `json:"outbox"`
}

// Runner runs jobs.
type Runner struct {
	pool    *pgxpool.Pool
	drainer *outbox.Drainer
	loc     *time.Location
	logger  *slog.Logger
	now     func() time.Time
}

// NewRunner returns a Runner. now may be nil, in which case time.Now is used.
func NewRunner(
	pool *pgxpool.Pool,
	drainer *outbox.Drainer,
	loc *time.Location,
	logger *slog.Logger,
	now func() time.Time,
) *Runner {
	if now == nil {
		now = time.Now
	}
	return &Runner{pool: pool, drainer: drainer, loc: loc, logger: logger, now: now}
}

// Run executes the named job. Jobs that queue messages also deliver them.
func (r *Runner) Run(ctx context.Context, job string) (Result, error) {
	var (
		result Result
		err    error
		drain  bool
	)
	q := db.New(r.pool)

	switch job {
	case DailySummary:
		result.Enqueued, err = r.dailySummaries(ctx, q)
		drain = true
	case WeeklyReminder:
		result.Enqueued, err = r.weeklyReminders(ctx, q)
		drain = true
	case DrainOutbox:
		drain = true
	case Cleanup:
		result.Deleted, err = r.cleanup(ctx, q)
	default:
		return Result{}, ErrUnknownJob
	}
	if err != nil {
		return result, fmt.Errorf("%s: %w", job, err)
	}
	if drain {
		result.Outbox, err = r.drainer.Drain(ctx, drainBudget)
		if err != nil {
			return result, fmt.Errorf("%s: %w", job, err)
		}
	}
	r.logger.Info("job finished",
		"job", job,
		"enqueued", result.Enqueued,
		"deleted", result.Deleted,
		"sent", result.Outbox.Sent,
		"failed", result.Outbox.Failed,
	)
	return result, nil
}

func (r *Runner) dailySummaries(ctx context.Context, q *db.Queries) (int, error) {
	now := r.now()
	day := ledger.Day(now, r.loc)
	users, err := q.SummaryRecipients(ctx, db.SummaryRecipientsParams{FromTime: day.From, ToTime: day.To})
	if err != nil {
		return 0, fmt.Errorf("recipients: %w", err)
	}

	enqueued := 0
	for _, userID := range users {
		totals, err := ledger.Totals(ctx, q, userID, day)
		if err != nil {
			return enqueued, err
		}
		key := fmt.Sprintf("daily:%d:%s", userID, day.From.Format(time.DateOnly))
		added, err := outbox.Enqueue(ctx, q, userID, format.Totals("Today", totals), key, now)
		if err != nil {
			return enqueued, err
		}
		if added {
			enqueued++
		}
	}
	return enqueued, nil
}

func (r *Runner) weeklyReminders(ctx context.Context, q *db.Queries) (int, error) {
	now := r.now()
	year, week := now.In(r.loc).ISOWeek()
	chats, err := q.GroupsWithOpenBalances(ctx)
	if err != nil {
		return 0, fmt.Errorf("groups: %w", err)
	}

	enqueued := 0
	for _, chatID := range chats {
		balances, err := ledger.Balances(ctx, q, chatID)
		if err != nil {
			return enqueued, err
		}
		text := format.Transfers("Open balances this week", ledger.Simplify(balances)) +
			"\n\nSend /balance to settle up."
		key := fmt.Sprintf("weekly:%d:%d-%02d", chatID, year, week)
		added, err := outbox.Enqueue(ctx, q, chatID, text, key, now)
		if err != nil {
			return enqueued, err
		}
		if added {
			enqueued++
		}
	}
	return enqueued, nil
}

func (r *Runner) cleanup(ctx context.Context, q *db.Queries) (int64, error) {
	now := r.now()
	updates, err := q.DeleteProcessedUpdates(ctx, now.Add(-updateRetention))
	if err != nil {
		return 0, fmt.Errorf("delete updates: %w", err)
	}
	messages, err := q.DeleteFinishedMessages(ctx, now.Add(-outboxRetention))
	if err != nil {
		return updates, fmt.Errorf("delete messages: %w", err)
	}
	return updates + messages, nil
}
