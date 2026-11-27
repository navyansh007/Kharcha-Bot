package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"

	"github.com/navyansh007/kharcha-bot/internal/cron"
)

const cronTimeout = 60 * time.Second

// JobRunner runs a scheduled job by name.
type JobRunner interface {
	Run(ctx context.Context, job string) (cron.Result, error)
}

// Cron configures the endpoint an external scheduler calls.
type Cron struct {
	Secret string
	Jobs   JobRunner
}

func (c Cron) handler(logger *slog.Logger, failures *rate.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || !secretEqual(token, c.Secret) {
			reject(w, failures, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cronTimeout)
		defer cancel()
		job := chi.URLParam(r, "job")
		result, err := c.Jobs.Run(ctx, job)
		if errors.Is(err, cron.ErrUnknownJob) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			logger.Error("job failed", "job", job, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			logger.Warn("write job result", "error", err)
		}
	}
}
