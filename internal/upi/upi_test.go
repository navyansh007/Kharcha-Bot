package upi

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var payment = Payment{VPA: "asha@okaxis", Name: "Asha K", Amount: 30000, Note: "hostel & dinner"}

func TestURI(t *testing.T) {
	want := "upi://pay?pa=asha@okaxis&pn=Asha%20K&am=300.00&cu=INR&tn=hostel%20%26%20dinner"
	if got := payment.URI(); got != want {
		t.Errorf("URI() = %q, want %q", got, want)
	}
}

func TestSignAndVerify(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	signer := NewSigner(strings.Repeat("k", 32), "https://bot.example/")

	link, err := signer.Link(payment, now)
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	token, ok := strings.CutPrefix(link, "https://bot.example/pay/")
	if !ok {
		t.Fatalf("Link() = %q", link)
	}

	got, err := signer.Verify(token, now.Add(time.Hour))
	if err != nil || got != payment {
		t.Fatalf("Verify() = %+v, %v", got, err)
	}

	tests := []struct {
		name  string
		token string
		at    time.Time
		key   string
		want  error
	}{
		{name: "expired", token: token, at: now.Add(TokenLifetime + time.Second), want: ErrExpiredToken},
		{name: "tampered", token: "x" + token, at: now, want: ErrInvalidToken},
		{name: "no signature", token: "abc", at: now, want: ErrInvalidToken},
		{name: "other key", token: token, at: now, key: strings.Repeat("z", 32), want: ErrInvalidToken},
	}
	for _, tt := range tests {
		verifier := signer
		if tt.key != "" {
			verifier = NewSigner(tt.key, "https://bot.example")
		}
		if _, err := verifier.Verify(tt.token, tt.at); !errors.Is(err, tt.want) {
			t.Errorf("%s: error = %v, want %v", tt.name, err, tt.want)
		}
	}
}
