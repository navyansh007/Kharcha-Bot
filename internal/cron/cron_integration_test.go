//go:build integration

package cron_test

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/navyansh007/kharcha-bot/internal/cron"
	"github.com/navyansh007/kharcha-bot/internal/outbox"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
	"github.com/navyansh007/kharcha-bot/internal/testdb"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, cleanup, err := testdb.Start(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	pool = p
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func newRunner(fake *telegram.Fake) *cron.Runner {
	logger := slog.New(slog.DiscardHandler)
	drainer := outbox.NewDrainer(pool, cron.TelegramSender{Client: fake}, logger)
	return cron.NewRunner(pool, drainer, time.UTC, logger, nil)
}

func exec(t *testing.T, query string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), query); err != nil {
		t.Fatal(err)
	}
}

func TestDailySummaryRunsOncePerDay(t *testing.T) {
	testdb.Reset(t, pool)
	exec(t, `INSERT INTO users (id, first_name, daily_summary) VALUES (1, 'Asha', true), (2, 'Ravi', false)`)
	exec(t, `INSERT INTO personal_expenses (user_id, amount, category, description)
		VALUES (1, 12000, 'food', 'chai'), (2, 5000, 'food', 'tea')`)
	fake := &telegram.Fake{}
	runner := newRunner(fake)

	for range 2 {
		if _, err := runner.Run(context.Background(), cron.DailySummary); err != nil {
			t.Fatal(err)
		}
	}

	if len(fake.Sent) != 1 || fake.Sent[0].ChatID != 1 || !strings.Contains(fake.Sent[0].Text, "₹120") {
		t.Errorf("sent = %+v", fake.Sent)
	}
}

func TestWeeklyReminderSkipsSettledGroups(t *testing.T) {
	testdb.Reset(t, pool)
	exec(t, `INSERT INTO users (id, first_name) VALUES (1, 'Asha'), (2, 'Ravi')`)
	exec(t, `INSERT INTO chats (id, kind) VALUES (-1, 'group'), (-2, 'group')`)
	exec(t, `INSERT INTO group_txns (id, chat_id, kind, created_by, description, total)
		VALUES (1, -1, 'expense', 1, 'dinner', 600), (2, -2, 'expense', 1, 'cab', 400),
		(3, -2, 'settlement', 2, 'settlement', 400)`)
	exec(t, `INSERT INTO group_entries (txn_id, user_id, amount)
		VALUES (1, 1, 300), (1, 2, -300), (2, 1, 400), (2, 2, -400), (3, 2, 400), (3, 1, -400)`)
	fake := &telegram.Fake{}

	if _, err := newRunner(fake).Run(context.Background(), cron.WeeklyReminder); err != nil {
		t.Fatal(err)
	}

	if len(fake.Sent) != 1 || fake.Sent[0].ChatID != -1 || !strings.Contains(fake.Sent[0].Text, "Ravi → Asha") {
		t.Errorf("sent = %+v", fake.Sent)
	}
}

func TestDatabaseRejectsUnbalancedTxn(t *testing.T) {
	testdb.Reset(t, pool)
	exec(t, `INSERT INTO users (id, first_name) VALUES (1, 'Asha')`)
	exec(t, `INSERT INTO chats (id, kind) VALUES (-1, 'group')`)
	exec(t, `INSERT INTO group_txns (id, chat_id, kind, created_by, description, total)
		VALUES (1, -1, 'expense', 1, 'dinner', 600)`)

	_, err := pool.Exec(context.Background(), `INSERT INTO group_entries (txn_id, user_id, amount) VALUES (1, 1, 300)`)
	if err == nil {
		t.Fatal("unbalanced entries were accepted")
	}
}

func TestCleanupAndUnknownJob(t *testing.T) {
	testdb.Reset(t, pool)
	exec(t, `INSERT INTO processed_updates (update_id, processed_at)
		VALUES (1, now() - interval '8 days'), (2, now())`)
	runner := newRunner(&telegram.Fake{})

	result, err := runner.Run(context.Background(), cron.Cleanup)
	if err != nil || result.Deleted != 1 {
		t.Errorf("result = %+v, error = %v", result, err)
	}
	if _, err := runner.Run(context.Background(), "nope"); !errors.Is(err, cron.ErrUnknownJob) {
		t.Errorf("error = %v", err)
	}
}
