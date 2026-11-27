//go:build integration

package bot_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/navyansh007/kharcha-bot/internal/bot"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

func TestFirstMessageAfterWakingIsAnnounced(t *testing.T) {
	h := newHarness(t)
	dm := private(asha)
	send := func(sentAt time.Time) string {
		u := h.message(asha, dm, "10 chai")
		u.Message.Date = sentAt.Unix()
		h.fake.Reset()
		h.handle(u)
		var texts []string
		for _, m := range h.fake.Sent {
			texts = append(texts, m.Text)
		}
		return strings.Join(texts, "\n")
	}
	beforeStart := h.now.Add(-time.Minute)

	if got := send(beforeStart); !strings.HasPrefix(got, "☀️") || !strings.Contains(got, "logged") {
		t.Errorf("first reply = %q, want the notice and then the reply", got)
	}
	if got := send(beforeStart); strings.Contains(got, "asleep") {
		t.Errorf("notice repeated: %q", got)
	}
	h.now = h.now.Add(time.Hour)
	if got := send(h.now); strings.Contains(got, "asleep") {
		t.Errorf("notice on a message sent while awake: %q", got)
	}
}

func TestFailureIsReportedOnce(t *testing.T) {
	closed, err := pgxpool.New(context.Background(), "postgres://nobody@127.0.0.1:1/none")
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	fake := &telegram.Fake{}
	b := bot.New(bot.Options{
		Pool:     closed,
		Client:   fake,
		Logger:   slog.New(slog.DiscardHandler),
		Location: time.UTC,
	})
	update := telegram.Update{
		UpdateID: 1,
		Message:  &telegram.Message{From: &asha, Chat: private(asha), Text: "10 chai"},
	}

	for range 3 {
		if err := b.HandleUpdate(context.Background(), update); err == nil {
			t.Fatal("HandleUpdate() succeeded without a database")
		}
	}
	if len(fake.Sent) != 1 || !strings.Contains(fake.Sent[0].Text, "went wrong") {
		t.Errorf("sent = %+v", fake.Sent)
	}
}
