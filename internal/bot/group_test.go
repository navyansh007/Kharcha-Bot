//go:build integration

package bot_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/ledger"
	"github.com/navyansh007/kharcha-bot/internal/store"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

var hostel = telegram.Chat{ID: -500, Type: "supergroup", Title: "Hostel"}

func newGroup(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	for _, u := range []telegram.User{asha, ravi, meera} {
		h.say(u, hostel, "/join")
	}
	return h
}

const (
	balanceSum   = "SELECT coalesce(sum(amount), 0) FROM group_entries"
	unbalanced   = "SELECT count(*) FROM (SELECT txn_id FROM group_entries GROUP BY txn_id HAVING sum(amount) <> 0) t"
	reversalRows = "SELECT count(*) FROM group_txns WHERE kind = 'reversal'"
)

func TestSplitAndSettleFlow(t *testing.T) {
	h := newGroup(t)

	golden(t, "split_equal", h.say(asha, hostel, "/split 900 dinner"))
	golden(t, "split_exact", h.say(ravi, hostel, "/split 900 cab @asha:300 @meera:600"))
	golden(t, "balance", h.say(asha, hostel, "/balance"))
	golden(t, "settle", h.say(meera, hostel, "/settle @asha 300"))

	if n := h.count(balanceSum); n != 0 {
		t.Errorf("group balance sum = %d", n)
	}
}

func TestSplitRejections(t *testing.T) {
	h := newGroup(t)
	tests := map[string]string{
		"/split 900 dinner @ravi:300 @meera:550": "Shares add up to ₹850, but the total is ₹900",
		"/split 900 dinner @stranger":            "@stranger hasn&#39;t joined yet",
		"/split 900 dinner @asha":                "no one to split with",
		"/split 900 dinner @ravi:450 @ravi:450":  "listed twice",
		"/settle @asha 10":                       "settle with yourself",
	}
	for command, want := range tests {
		if got := h.say(asha, hostel, command); !strings.Contains(got, want) {
			t.Errorf("%s: reply = %q, want it to contain %q", command, got, want)
		}
	}
	if n := h.count("SELECT count(*) FROM group_txns"); n != 0 {
		t.Errorf("rejected commands wrote %d txns", n)
	}
}

func TestConcurrentSplitsKeepBalancesAtZero(t *testing.T) {
	h := newGroup(t)
	const n = 200

	var wg sync.WaitGroup
	for i := range n {
		payer := []telegram.User{asha, ravi, meera}[i%3]
		u := h.message(payer, hostel, "/split 100 snacks")
		wg.Go(func() { h.handle(u) })
	}
	wg.Wait()

	if got := h.count("SELECT count(*) FROM group_txns"); got != n {
		t.Errorf("txns = %d, want %d", got, n)
	}
	if got := h.count("SELECT count(*) FROM group_entries"); got != 3*n {
		t.Errorf("entries = %d, want %d", got, 3*n)
	}
	if got := h.count(balanceSum); got != 0 {
		t.Errorf("group balance sum = %d", got)
	}
	if got := h.count(unbalanced); got != 0 {
		t.Errorf("unbalanced txns = %d", got)
	}
}

func TestDoubleUndoPressReversesOnce(t *testing.T) {
	h := newGroup(t)
	h.say(asha, hostel, "/split 900 dinner")

	var wg sync.WaitGroup
	for range 2 {
		u := h.press(asha, hostel, "undo:1")
		wg.Go(func() { h.handle(u) })
	}
	wg.Wait()

	if n := h.count(reversalRows); n != 1 {
		t.Errorf("reversals = %d, want 1", n)
	}
}

func TestConcurrentUndoReliesOnUniqueConstraint(t *testing.T) {
	h := newGroup(t)
	h.say(asha, hostel, "/split 900 dinner")

	const n = 10
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			var undoErr error
			err := store.InTx(context.Background(), pool, func(q *db.Queries) error {
				_, undoErr = ledger.UndoTxn(context.Background(), q, hostel.ID, asha.ID, 1, h.now)
				return nil
			})
			errs[i] = errors.Join(err, undoErr)
		})
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case !errors.Is(err, ledger.ErrAlreadyUndone):
			t.Errorf("unexpected error: %v", err)
		}
	}
	if succeeded != 1 || h.count(reversalRows) != 1 {
		t.Errorf("succeeded = %d, reversals = %d, want 1", succeeded, h.count(reversalRows))
	}
}

func TestOnlyCreatorCanUndo(t *testing.T) {
	h := newGroup(t)
	h.say(asha, hostel, "/split 900 dinner")

	h.fake.Reset()
	h.handle(h.press(ravi, hostel, "undo:1"))

	if n := h.count(reversalRows); n != 0 {
		t.Errorf("reversals = %d, want 0", n)
	}
	if len(h.fake.Answered) != 1 || !strings.Contains(h.fake.Answered[0].Text, "Only the person") {
		t.Errorf("answers = %+v", h.fake.Answered)
	}
}

func TestSettleButtonRecordsOnce(t *testing.T) {
	h := newGroup(t)
	h.say(asha, hostel, "/split 900 dinner")
	h.say(asha, hostel, "/balance")
	data := h.fake.Sent[0].ReplyMarkup.InlineKeyboard[0][0].CallbackData

	h.handle(h.press(meera, hostel, "settle:102:101:30000"))
	h.handle(h.press(asha, hostel, data))
	h.handle(h.press(asha, hostel, data))

	if n := h.count("SELECT count(*) FROM group_txns WHERE kind = 'settlement'"); n != 1 {
		t.Errorf("settlements = %d, want 1", n)
	}
}

func TestMemberWhoLeftIsNotSplitWith(t *testing.T) {
	h := newGroup(t)
	left := h.message(meera, hostel, "")
	left.Message.LeftChatMember = &meera
	h.handle(left)

	if got := h.say(asha, hostel, "/split 900 dinner"); !strings.Contains(got, "split 2 ways") {
		t.Errorf("reply = %q", got)
	}
}
