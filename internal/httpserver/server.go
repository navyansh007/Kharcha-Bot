// Package httpserver exposes the bot's HTTP routes and middleware.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/navyansh007/kharcha-bot/internal/upi"
)

const (
	maxBodyBytes    = 1 << 20
	shutdownTimeout = 15 * time.Second
	readyTimeout    = 3 * time.Second
)

// Options selects the routes served by NewHandler. Nil routes are not registered.
type Options struct {
	Logger  *slog.Logger
	Webhook *Webhook
	Pay     *upi.Signer
	Cron    *Cron
	Ready   Pinger
	Metrics http.Handler
}

// Pinger reports whether a dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// NewHandler builds the router with the common middleware and routes.
func NewHandler(opts Options) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(logRequests(opts.Logger))
	r.Use(recoverPanics(opts.Logger))
	r.Use(middleware.RequestSize(maxBodyBytes))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	if opts.Ready != nil {
		r.Get("/readyz", readyHandler(opts.Ready, opts.Logger))
	}
	if opts.Metrics != nil {
		r.Method(http.MethodGet, "/metrics", opts.Metrics)
	}
	if wh := opts.Webhook; wh != nil {
		r.Post("/webhook/"+wh.PathToken, wh.handler(opts.Logger))
	}
	if opts.Pay != nil {
		r.Get("/pay/{token}", payHandler(opts.Pay, opts.Logger))
	}
	if c := opts.Cron; c != nil {
		r.Post("/cron/{job}", c.handler(opts.Logger))
	}
	return r
}

func readyHandler(db Pinger, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			logger.Warn("not ready", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ready"))
	}
}

// Run serves handler on addr until ctx is cancelled, then shuts down gracefully.
func Run(ctx context.Context, addr string, handler http.Handler, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      70 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
