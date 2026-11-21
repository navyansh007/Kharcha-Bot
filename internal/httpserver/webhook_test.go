package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

type stubUpdates struct {
	calls int
	err   error
}

func (s *stubUpdates) HandleUpdate(context.Context, telegram.Update) error {
	s.calls++
	return s.err
}

func TestWebhook(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	const path = "/webhook/pathtoken1"

	tests := []struct {
		name      string
		path      string
		secret    string
		body      string
		err       error
		wantCode  int
		wantCalls int
	}{
		{name: "ok", path: path, secret: secret, body: `{"update_id":1}`, wantCode: 200, wantCalls: 1},
		{name: "wrong secret", path: path, secret: "nope", body: `{"update_id":1}`, wantCode: 401},
		{name: "no secret", path: path, body: `{"update_id":1}`, wantCode: 401},
		{name: "wrong path", path: "/webhook/other", secret: secret, body: `{}`, wantCode: 404},
		{name: "bad json", path: path, secret: secret, body: `{`, wantCode: 400},
		{
			name: "too large", path: path, secret: secret,
			body:     `{"x":"` + strings.Repeat("a", maxBodyBytes) + `"}`,
			wantCode: 413,
		},
		{
			name: "handler error", path: path, secret: secret, body: `{"update_id":1}`,
			err: errors.New("database down"), wantCode: 500, wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates := &stubUpdates{err: tt.err}
			handler := NewHandler(Options{
				Logger:  slog.New(slog.DiscardHandler),
				Webhook: &Webhook{PathToken: "pathtoken1", Secret: secret, Updates: updates},
			})
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			if tt.secret != "" {
				req.Header.Set(webhookHeader, tt.secret)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if updates.calls != tt.wantCalls {
				t.Errorf("handler calls = %d, want %d", updates.calls, tt.wantCalls)
			}
		})
	}
}
