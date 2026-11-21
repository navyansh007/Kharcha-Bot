package ledger

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/money"
)

func TestEqualSharesOrdersRemainderByUserID(t *testing.T) {
	got := EqualShares(1000, []int64{30, 10, 20, 10})
	want := []Share{{10, 334}, {20, 333}, {30, 333}}
	if !slices.Equal(got, want) {
		t.Errorf("EqualShares() = %v, want %v", got, want)
	}
}

func TestSplitEntries(t *testing.T) {
	tests := []struct {
		name    string
		shares  []Share
		want    []Entry
		wantErr error
	}{
		{
			name:   "payer included",
			shares: []Share{{1, 30000}, {2, 30000}, {3, 30000}},
			want:   []Entry{{1, 60000}, {2, -30000}, {3, -30000}},
		},
		{
			name:   "payer excluded",
			shares: []Share{{2, 30000}, {3, 60000}},
			want:   []Entry{{1, 90000}, {2, -30000}, {3, -60000}},
		},
		{name: "only the payer", shares: []Share{{1, 90000}}, wantErr: ErrNoOtherParticipant},
		{name: "listed twice", shares: []Share{{2, 45000}, {2, 45000}}, wantErr: ErrDuplicateShare},
	}
	for _, tt := range tests {
		got, err := splitEntries(SplitInput{PayerID: 1, Total: 90000, Shares: tt.shares})
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("%s: error = %v, want %v", tt.name, err, tt.wantErr)
			continue
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: entries = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSplitEntriesShareMismatch(t *testing.T) {
	_, err := splitEntries(SplitInput{PayerID: 1, Total: 90000, Shares: []Share{{2, 85000}}})
	var mismatch *ShareMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("error = %v, want *ShareMismatchError", err)
	}
	if mismatch.Sum != 85000 || mismatch.Total != 90000 {
		t.Errorf("mismatch = %+v", mismatch)
	}
}

func TestSimplifySettlesEveryBalance(t *testing.T) {
	for range 2000 {
		n := rand.IntN(12) + 2
		balances := make([]Balance, n)
		var sum money.Paise
		for i := range n - 1 {
			amount := money.Paise(rand.Int64N(20001) - 10000)
			balances[i] = Balance{UserID: int64(i), Amount: amount}
			sum += amount
		}
		balances[n-1] = Balance{UserID: int64(n - 1), Amount: -sum}

		transfers := Simplify(balances)
		if len(transfers) > n-1 {
			t.Fatalf("%d transfers for %d members", len(transfers), n)
		}
		left := make([]money.Paise, n)
		for i, b := range balances {
			left[i] = b.Amount
		}
		for _, tr := range transfers {
			if tr.Amount <= 0 {
				t.Fatalf("non-positive transfer %+v", tr)
			}
			left[tr.From.UserID] += tr.Amount
			left[tr.To.UserID] -= tr.Amount
		}
		for i, amount := range left {
			if amount != 0 {
				t.Fatalf("balances %v: member %d left with %d", balances, i, amount)
			}
		}
	}
}
