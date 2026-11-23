//go:build integration

package outbox_test

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/navyansh007/kharcha-bot/internal/outbox"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
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

type recorder struct {
	mu   sync.Mutex
	sent map[int64]int
	err  error
}

func (r *recorder) Send(_ context.Context, chatID int64, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.sent[chatID]++
	return nil
}

func enqueue(t *testing.T, chatID int64, key string) bool {
	t.Helper()
	added, err := outbox.Enqueue(context.Background(), db.New(pool), chatID, "hello", key, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return added
}

func status(t *testing.T) (status string, attempts int) {
	t.Helper()
	err := pool.QueryRow(context.Background(), "SELECT status, attempts FROM outbox").Scan(&status, &attempts)
	if err != nil {
		t.Fatal(err)
	}
	return status, attempts
}

func drainer(r *recorder) *outbox.Drainer {
	return outbox.NewDrainer(pool, r, slog.New(slog.DiscardHandler))
}

func TestEnqueueDeduplicates(t *testing.T) {
	testdb.Reset(t, pool)
	if !enqueue(t, 1, "daily:1") || enqueue(t, 1, "daily:1") {
		t.Fatal("second enqueue with the same key was not ignored")
	}
}

func TestConcurrentDrainsSendEachMessageOnce(t *testing.T) {
	testdb.Reset(t, pool)
	for i := range 20 {
		enqueue(t, int64(i+1), time.Duration(i).String())
	}
	r := &recorder{sent: map[int64]int{}}

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if _, err := drainer(r).Drain(context.Background(), 10*time.Second); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	if len(r.sent) != 20 {
		t.Errorf("chats reached = %d, want 20", len(r.sent))
	}
	for chatID, n := range r.sent {
		if n != 1 {
			t.Errorf("chat %d received %d messages", chatID, n)
		}
	}
}

func TestDrainFailureHandling(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name         string
		err          error
		wantStatus   string
		wantAttempts int
	}{
		{"temporary", boom, "pending", 1},
		{"rate limited", &outbox.SendError{Err: boom, RetryAfter: time.Minute}, "pending", 0},
		{"blocked", &outbox.SendError{Err: boom, Blocked: true}, "failed", 1},
		{"permanent", &outbox.SendError{Err: boom, Permanent: true}, "failed", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testdb.Reset(t, pool)
			enqueue(t, 7, "k")
			if _, err := drainer(&recorder{err: tt.err}).Drain(context.Background(), time.Second); err != nil {
				t.Fatal(err)
			}
			gotStatus, gotAttempts := status(t)
			if gotStatus != tt.wantStatus || gotAttempts != tt.wantAttempts {
				t.Errorf("status = %s, attempts = %d", gotStatus, gotAttempts)
			}
		})
	}
}
