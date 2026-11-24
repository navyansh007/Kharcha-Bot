package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := NewHandler(Options{Logger: slog.New(slog.DiscardHandler)})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestReadyz(t *testing.T) {
	for err, want := range map[error]int{nil: http.StatusOK, errors.New("down"): http.StatusServiceUnavailable} {
		h := NewHandler(Options{Logger: slog.New(slog.DiscardHandler), Ready: stubPinger{err}})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != want {
			t.Errorf("ping error %v: status = %d, want %d", err, rec.Code, want)
		}
	}
}
