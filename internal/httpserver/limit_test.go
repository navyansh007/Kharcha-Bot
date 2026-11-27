package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailedAuthIsLimitedButValidCallsPass(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	handler := NewHandler(Options{
		Logger:  slog.New(slog.DiscardHandler),
		Webhook: &Webhook{PathToken: "pathtoken1", Secret: secret, Updates: &stubUpdates{}},
	})
	post := func(value string) int {
		req := httptest.NewRequest(http.MethodPost, "/webhook/pathtoken1", strings.NewReader(`{"update_id":1}`))
		req.Header.Set(webhookHeader, value)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	last := 0
	for range 50 {
		last = post("wrong")
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("status after many failures = %d, want 429", last)
	}
	if code := post(secret); code != http.StatusOK {
		t.Errorf("valid delivery status = %d, want 200", code)
	}
}

func TestReadyzIsLimited(t *testing.T) {
	handler := NewHandler(Options{Logger: slog.New(slog.DiscardHandler), Ready: stubPinger{}})
	codes := map[int]int{}
	for range 10 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		codes[rec.Code]++
	}
	if codes[http.StatusOK] != readyBurst || codes[http.StatusTooManyRequests] != 10-readyBurst {
		t.Errorf("status counts = %v", codes)
	}
}
