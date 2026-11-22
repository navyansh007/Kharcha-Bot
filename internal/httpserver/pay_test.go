package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/upi"
)

func TestPayPage(t *testing.T) {
	signer := upi.NewSigner(strings.Repeat("k", 32), "https://bot.example")
	handler := NewHandler(Options{Logger: slog.New(slog.DiscardHandler), Pay: signer})
	payment := upi.Payment{VPA: "asha@okaxis", Name: `<script>alert(1)</script>`, Amount: 30000, Note: "dinner"}

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}
	link, err := signer.Link(payment, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	rec := get(strings.TrimPrefix(link, "https://bot.example"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `href="upi://pay?pa=asha@okaxis`) {
		t.Errorf("page has no upi link:\n%s", body)
	}
	if strings.Contains(body, "<script>alert") || strings.Contains(body, "ZgotmplZ") {
		t.Errorf("page is not escaped correctly:\n%s", body)
	}

	if rec := get("/pay/forged.token"); rec.Code != http.StatusNotFound {
		t.Errorf("forged token status = %d", rec.Code)
	}
	expired, err := signer.Link(payment, time.Now().Add(-2*upi.TokenLifetime))
	if err != nil {
		t.Fatal(err)
	}
	if rec := get(strings.TrimPrefix(expired, "https://bot.example")); rec.Code != http.StatusGone {
		t.Errorf("expired token status = %d", rec.Code)
	}
}
