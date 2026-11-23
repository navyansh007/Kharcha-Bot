package budget

import (
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/money"
)

func TestCrossed(t *testing.T) {
	tests := []struct {
		spent     money.Paise
		threshold int16
		want      bool
	}{
		{79_99, 80, false},
		{80_00, 80, true},
		{99_99, 100, false},
		{100_00, 100, true},
	}
	for _, tt := range tests {
		if got := crossed(tt.spent, 100_00, tt.threshold); got != tt.want {
			t.Errorf("crossed(%d, %d) = %v, want %v", tt.spent, tt.threshold, got, tt.want)
		}
	}
}
