package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
)

func logRequests(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"route", chi.RouteContext(r.Context()).RoutePattern(),
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func limit(limiter *rate.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func reject(w http.ResponseWriter, failures *rate.Limiter, status int) {
	if !failures.Allow() {
		status = http.StatusTooManyRequests
	}
	w.WriteHeader(status)
}

func recoverPanics(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					logger.Error("panic in handler",
						"request_id", middleware.GetReqID(r.Context()),
						"panic", v,
					)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
