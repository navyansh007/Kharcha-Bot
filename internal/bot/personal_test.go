//go:build integration

package bot_test

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPersonalExpenseFlow(t *testing.T) {
	h := newHarness(t)
	dm := private(asha)

	golden(t, "logged", h.say(asha, dm, "120 chai"))
	h.say(asha, dm, "450 uber to airport")
	h.say(asha, dm, "900 shoes #gifts")
	golden(t, "today", h.say(asha, dm, "/today"))

	h.say(asha, dm, "/undo")
	if got := h.say(asha, dm, "/today"); strings.Contains(got, "gifts") {
		t.Errorf("undone expense still listed: %q", got)
	}
}

func TestUndoWindow(t *testing.T) {
	h := newHarness(t)
	dm := private(asha)
	h.say(asha, dm, "120 chai")

	h.now = h.now.Add(11 * time.Minute)
	if got := h.say(asha, dm, "/undo"); !strings.Contains(got, "Too late") {
		t.Errorf("reply = %q", got)
	}
}

func TestUndoButtonIsOwnerOnly(t *testing.T) {
	h := newHarness(t)
	h.say(asha, private(asha), "120 chai")

	h.handle(h.press(ravi, private(ravi), "pundo:1"))
	if n := h.count("SELECT count(*) FROM personal_expenses WHERE deleted_at IS NULL"); n != 1 {
		t.Fatalf("live expenses = %d, want 1", n)
	}

	h.handle(h.press(asha, private(asha), "pundo:1"))
	if n := h.count("SELECT count(*) FROM personal_expenses WHERE deleted_at IS NULL"); n != 0 {
		t.Fatalf("live expenses = %d, want 0", n)
	}
}

func TestDuplicateDeliveryLogsOnce(t *testing.T) {
	h := newHarness(t)
	u := h.message(asha, private(asha), "120 chai")

	for range 5 {
		h.handle(u)
	}

	if n := h.count("SELECT count(*) FROM personal_expenses"); n != 1 {
		t.Errorf("expenses = %d, want 1", n)
	}
	if n := len(h.fake.Sent); n != 1 {
		t.Errorf("replies = %d, want 1", n)
	}
}

func TestConcurrentDuplicateDeliveryLogsOnce(t *testing.T) {
	h := newHarness(t)
	u := h.message(asha, private(asha), "120 chai")

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { h.handle(u) })
	}
	wg.Wait()

	if n := h.count("SELECT count(*) FROM personal_expenses"); n != 1 {
		t.Errorf("expenses = %d, want 1", n)
	}
}
