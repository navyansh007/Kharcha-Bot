//go:build integration

package bot_test

import (
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

func TestSummaryAndBlocking(t *testing.T) {
	h := newHarness(t)
	h.say(asha, private(asha), "/summary on")
	if n := h.count("SELECT count(*) FROM users WHERE daily_summary"); n != 1 {
		t.Fatalf("opted in users = %d", n)
	}

	h.handle(telegram.Update{
		UpdateID: h.updateID.Add(1),
		MyChatMember: &telegram.ChatMemberUpdated{
			Chat:          private(asha),
			From:          asha,
			NewChatMember: telegram.ChatMember{Status: "kicked"},
		},
	})
	if n := h.count("SELECT count(*) FROM users WHERE blocked_at IS NOT NULL"); n != 1 {
		t.Fatalf("blocked users = %d", n)
	}

	h.say(asha, private(asha), "/start")
	if n := h.count("SELECT count(*) FROM users WHERE blocked_at IS NOT NULL"); n != 0 {
		t.Errorf("user is still marked blocked after writing again")
	}
}
