//go:build integration

package bot_test

import (
	"strings"
	"testing"
)

func TestBudgetWarnsOncePerThreshold(t *testing.T) {
	h := newHarness(t)
	dm := private(asha)
	h.say(asha, dm, "/budget food 1000")

	steps := []struct {
		message string
		want    string
	}{
		{"700 lunch", ""},
		{"100 chai", "80%"},
		{"50 chai", ""},
		{"200 dinner", "used up"},
		{"10 chai", ""},
	}
	for _, step := range steps {
		got := h.say(asha, dm, step.message)
		warned := strings.Contains(got, "budget")
		if warned != (step.want != "") || !strings.Contains(got, step.want) {
			t.Errorf("%s: reply = %q, want warning %q", step.message, got, step.want)
		}
	}
}
