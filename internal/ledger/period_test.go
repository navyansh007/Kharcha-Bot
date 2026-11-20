package ledger

import (
	"testing"
	"time"
)

func TestPeriods(t *testing.T) {
	ist := time.FixedZone("IST", 5*60*60+30*60)
	now := time.Date(2026, 3, 1, 20, 0, 0, 0, time.UTC)
	date := func(month time.Month, day int) time.Time {
		return time.Date(2026, month, day, 0, 0, 0, 0, ist)
	}

	tests := []struct {
		name string
		got  Period
		want Period
	}{
		{"day", Day(now, ist), Period{date(3, 2), date(3, 3)}},
		{"week", Week(now, ist), Period{date(3, 2), date(3, 9)}},
		{"week on sunday", Week(date(3, 1), ist), Period{date(2, 23), date(3, 2)}},
		{"month", Month(now, ist), Period{date(3, 1), date(4, 1)}},
	}
	for _, tt := range tests {
		if !tt.got.From.Equal(tt.want.From) || !tt.got.To.Equal(tt.want.To) {
			t.Errorf("%s = %v to %v, want %v to %v", tt.name, tt.got.From, tt.got.To, tt.want.From, tt.want.To)
		}
	}
}
