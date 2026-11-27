package bot

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/ratelimit"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

const (
	wokeUpText = "☀️ I was asleep, so that took a moment. I'm ready now."
	issueText  = "⚠️ Something went wrong on my side. I'll try that again shortly, " +
		"so please don't send it twice yet."
)

type notices struct {
	startedAt time.Time
	failures  *ratelimit.Keyed

	mu    sync.Mutex
	woken map[int64]bool
}

func newNotices(startedAt time.Time) *notices {
	return &notices{
		startedAt: startedAt,
		failures:  ratelimit.NewKeyed(10*time.Minute, 1),
		woken:     map[int64]bool{},
	}
}

// firstSinceWake reports whether message was waiting for this process to start
// and is the first such message from its chat. A sleeping host runs no code, so
// this is the earliest the sender can be told.
func (n *notices) firstSinceWake(message *telegram.Message) bool {
	if message == nil || message.Date <= 0 || message.Date >= n.startedAt.Unix() {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.woken[message.Chat.ID] {
		return false
	}
	n.woken[message.Chat.ID] = true
	return true
}

func (b *Bot) announceWake(update telegram.Update, result *response) {
	if len(result.messages)+len(result.documents) == 0 || !b.notices.firstSinceWake(update.Message) {
		return
	}
	notice := telegram.SendMessageParams{ChatID: result.chatID, Text: wokeUpText}
	result.messages = slices.Insert(result.messages, 0, notice)
}

func (b *Bot) reportFailure(ctx context.Context, update telegram.Update) {
	message := update.Message
	if message == nil || !strings.HasPrefix(message.Text, "/") && message.Chat.Type != chatPrivate {
		return
	}
	if !b.notices.failures.Allow(message.Chat.ID, b.now()) {
		return
	}
	result := response{chatID: message.Chat.ID}
	result.reply(issueText)
	b.deliver(context.WithoutCancel(ctx), result)
}
