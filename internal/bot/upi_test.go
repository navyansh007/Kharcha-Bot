//go:build integration

package bot_test

import (
	"strings"
	"testing"
)

func TestBalanceOffersUPIPayment(t *testing.T) {
	h := newGroup(t)
	h.say(asha, hostel, "/setupi asha@okaxis")
	h.say(asha, hostel, "/split 900 dinner @ravi")
	h.say(asha, hostel, "/balance")

	keyboard := h.fake.Sent[0].ReplyMarkup.InlineKeyboard
	if len(keyboard) != 2 {
		t.Fatalf("keyboard = %+v, want a pay row and a settle row", keyboard)
	}
	if url := keyboard[0][0].URL; !strings.HasPrefix(url, "https://bot.example/pay/") {
		t.Errorf("pay url = %q", url)
	}
}
