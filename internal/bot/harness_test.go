//go:build integration

package bot_test

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/navyansh007/kharcha-bot/internal/bot"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
	"github.com/navyansh007/kharcha-bot/internal/testdb"
)

var update = flag.Bool("update", false, "rewrite golden files")

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	flag.Parse()
	p, cleanup, err := testdb.Start(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	pool = p
	code := m.Run()
	cleanup()
	os.Exit(code)
}

type harness struct {
	t        *testing.T
	bot      *bot.Bot
	fake     *telegram.Fake
	now      time.Time
	updateID atomic.Int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	testdb.Reset(t, pool)
	h := &harness{
		t:    t,
		fake: &telegram.Fake{},
		now:  time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
	}
	h.bot = bot.New(bot.Options{
		Pool:        pool,
		Client:      h.fake,
		Logger:      slog.New(slog.DiscardHandler),
		Location:    time.FixedZone("IST", 5*60*60+30*60),
		BotUsername: "KharchaBot",
		MaxExpense:  10_000_000,
		Now:         func() time.Time { return h.now },
	})
	return h
}

func (h *harness) handle(u telegram.Update) {
	h.t.Helper()
	if err := h.bot.HandleUpdate(context.Background(), u); err != nil {
		h.t.Errorf("HandleUpdate() error = %v", err)
	}
}

func (h *harness) message(from telegram.User, chat telegram.Chat, text string) telegram.Update {
	var entities []telegram.MessageEntity
	offset := 0
	for _, word := range strings.SplitAfter(text, " ") {
		trimmed := strings.TrimSpace(word)
		if strings.HasPrefix(trimmed, "@") {
			name, _, _ := strings.Cut(trimmed, ":")
			entities = append(entities, telegram.MessageEntity{
				Type: "mention", Offset: offset, Length: len(utf16.Encode([]rune(name))),
			})
		}
		offset += len(utf16.Encode([]rune(word)))
	}
	id := h.updateID.Add(1)
	return telegram.Update{
		UpdateID: id,
		Message:  &telegram.Message{MessageID: id, From: &from, Chat: chat, Text: text, Entities: entities},
	}
}

func (h *harness) say(from telegram.User, chat telegram.Chat, text string) string {
	h.t.Helper()
	h.fake.Reset()
	h.handle(h.message(from, chat, text))
	var replies []string
	for _, m := range h.fake.Sent {
		replies = append(replies, m.Text)
	}
	return strings.Join(replies, "\n---\n")
}

func (h *harness) press(from telegram.User, chat telegram.Chat, data string) telegram.Update {
	return telegram.Update{
		UpdateID: h.updateID.Add(1),
		CallbackQuery: &telegram.CallbackQuery{
			ID:      "callback",
			From:    from,
			Data:    data,
			Message: &telegram.Message{MessageID: 1, Chat: chat, Text: "original"},
		},
	}
}

func (h *harness) count(query string) int {
	h.t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query).Scan(&n); err != nil {
		h.t.Fatalf("count: %v", err)
	}
	return n
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got+"\n"), 0o600); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if got+"\n" != string(want) {
		t.Errorf("reply mismatch for %s\ngot:\n%s\nwant:\n%s", name, got, want)
	}
}

func private(u telegram.User) telegram.Chat {
	return telegram.Chat{ID: u.ID, Type: "private"}
}

var (
	asha  = telegram.User{ID: 101, FirstName: "Asha", Username: "asha"}
	ravi  = telegram.User{ID: 102, FirstName: "Ravi", Username: "ravi"}
	meera = telegram.User{ID: 103, FirstName: "Meera", Username: "meera"}
)
