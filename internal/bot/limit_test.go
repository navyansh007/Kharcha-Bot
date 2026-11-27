//go:build integration

package bot_test

import (
	"strings"
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/ratelimit"
)

func TestFloodIsRefusedWithOneNotice(t *testing.T) {
	h := newHarnessWithLimits(t, ratelimit.NewGuard())
	dm := private(asha)

	for range 50 {
		h.handle(h.message(asha, dm, "10 chai"))
	}

	if n := h.count("SELECT count(*) FROM personal_expenses"); n != 10 {
		t.Errorf("expenses = %d, want the burst of 10", n)
	}
	if n := h.count("SELECT count(*) FROM processed_updates"); n != 10 {
		t.Errorf("refused updates reached the database: %d rows", n)
	}
	notices := 0
	for _, m := range h.fake.Sent {
		if strings.Contains(m.Text, "too much") {
			notices++
		}
	}
	if notices != 1 {
		t.Errorf("notices = %d, want 1", notices)
	}
}
