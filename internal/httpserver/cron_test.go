package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/cron"
)

type stubJobs struct{}

func (stubJobs) Run(_ context.Context, job string) (cron.Result, error) {
	if job != cron.Cleanup {
		return cron.Result{}, cron.ErrUnknownJob
	}
	return cron.Result{Deleted: 3}, nil
}

func TestCronAuth(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	handler := NewHandler(Options{
		Logger: slog.New(slog.DiscardHandler),
		Cron:   &Cron{Secret: secret, Jobs: stubJobs{}},
	})

	tests := []struct {
		path, auth string
		want       int
	}{
		{"/cron/cleanup", "Bearer " + secret, http.StatusOK},
		{"/cron/cleanup", "Bearer wrong", http.StatusUnauthorized},
		{"/cron/cleanup", secret, http.StatusUnauthorized},
		{"/cron/cleanup", "", http.StatusUnauthorized},
		{"/cron/nope", "Bearer " + secret, http.StatusNotFound},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodPost, tt.path, nil)
		req.Header.Set("Authorization", tt.auth)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s with %q: status = %d, want %d", tt.path, tt.auth, rec.Code, tt.want)
		}
	}
}
