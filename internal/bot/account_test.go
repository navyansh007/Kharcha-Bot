//go:build integration

package bot_test

import (
	"strings"
	"testing"
)

func TestExport(t *testing.T) {
	h := newGroup(t)
	dm := private(asha)
	h.say(asha, dm, "120 chai")
	h.say(asha, hostel, "/split 900 dinner")
	h.say(asha, dm, "/export")

	if len(h.fake.Documents) != 1 {
		t.Fatalf("documents = %d", len(h.fake.Documents))
	}
	content := string(h.fake.Documents[0].Content)
	for _, want := range []string{"personal,,food,chai,120.00", "expense,Hostel,,dinner,600.00"} {
		if !strings.Contains(content, want) {
			t.Errorf("export is missing %q:\n%s", want, content)
		}
	}
}

func TestDeleteMeKeepsTheSharedLedger(t *testing.T) {
	h := newGroup(t)
	dm := private(asha)
	h.say(asha, dm, "120 chai")
	h.say(asha, dm, "/budget food 100")
	h.say(asha, dm, "/setupi asha@okaxis")
	h.say(asha, hostel, "/split 900 dinner")

	h.say(asha, dm, "/deleteme")
	if n := h.count("SELECT count(*) FROM personal_expenses"); n != 1 {
		t.Fatalf("data was deleted without confirmation")
	}
	h.say(asha, dm, "/deleteme confirm")

	for _, table := range []string{"personal_expenses", "budgets", "budget_alerts"} {
		if n := h.count("SELECT count(*) FROM " + table); n != 0 {
			t.Errorf("%s still has %d rows", table, n)
		}
	}
	if n := h.count("SELECT count(*) FROM users WHERE id = 101 AND username IS NULL AND upi_vpa IS NULL"); n != 1 {
		t.Errorf("user was not anonymised")
	}
	if n := h.count(balanceSum); n != 0 || h.count("SELECT count(*) FROM group_entries") != 3 {
		t.Errorf("group ledger changed")
	}
}
