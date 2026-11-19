package money

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    Paise
		wantErr bool
	}{
		{in: "120", want: 12000},
		{in: "99.5", want: 9950},
		{in: "99.05", want: 9905},
		{in: "0", want: 0},
		{in: "1.5k", want: 150000},
		{in: "2K", want: 200000},
		{in: "", wantErr: true},
		{in: "-5", wantErr: true},
		{in: "+5", wantErr: true},
		{in: "1.234", wantErr: true},
		{in: "1.", wantErr: true},
		{in: ".5", wantErr: true},
		{in: "1,000", wantErr: true},
		{in: "chai", wantErr: true},
		{in: "9999999999999", wantErr: true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		in         Paise
		wantString string
		wantRupees string
	}{
		{0, "₹0", "0.00"},
		{5, "₹0.05", "0.05"},
		{90000, "₹900", "900.00"},
		{123450, "₹1,234.50", "1234.50"},
		{1234567800, "₹1,23,45,678", "12345678.00"},
		{-30000, "-₹300", "-300.00"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.wantString {
			t.Errorf("Paise(%d).String() = %q, want %q", int64(tt.in), got, tt.wantString)
		}
		if got := tt.in.Rupees(); got != tt.wantRupees {
			t.Errorf("Paise(%d).Rupees() = %q, want %q", int64(tt.in), got, tt.wantRupees)
		}
	}
}

func TestSplitEqual(t *testing.T) {
	if got, want := SplitEqual(1000, 3), []Paise{334, 333, 333}; !slices.Equal(got, want) {
		t.Errorf("SplitEqual(1000, 3) = %v, want %v", got, want)
	}
	if got := SplitEqual(1000, 0); got != nil {
		t.Errorf("SplitEqual(1000, 0) = %v, want nil", got)
	}
}

func TestSplitEqualAlwaysSumsToTotal(t *testing.T) {
	for range 1000 {
		total := Paise(rand.Int64N(10_000_000) + 1)
		n := rand.IntN(50) + 1
		shares := SplitEqual(total, n)
		if got := Sum(shares); got != total {
			t.Fatalf("sum of SplitEqual(%d, %d) = %d", total, n, got)
		}
		if slices.Max(shares)-slices.Min(shares) > 1 {
			t.Fatalf("SplitEqual(%d, %d) = %v, shares differ by more than 1", total, n, shares)
		}
	}
}
