package httpserver

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

const (
	webhookHeader  = "X-Telegram-Bot-Api-Secret-Token"
	webhookTimeout = 25 * time.Second
)

// UpdateHandler processes one Telegram update.
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, update telegram.Update) error
}

// Webhook configures the endpoint Telegram delivers updates to.
type Webhook struct {
	PathToken string
	Secret    string
	Updates   UpdateHandler
}

func (wh Webhook) handler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !secretEqual(chi.URLParam(r, "token"), wh.PathToken) {
			http.NotFound(w, r)
			return
		}
		if !secretEqual(r.Header.Get(webhookHeader), wh.Secret) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var update telegram.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), webhookTimeout)
		defer cancel()
		if err := wh.Updates.HandleUpdate(ctx, update); err != nil {
			logger.Error("webhook update failed", "update_id", update.UpdateID, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func secretEqual(got, want string) bool {
	gotSum := sha256.Sum256([]byte(got))
	wantSum := sha256.Sum256([]byte(want))
	return want != "" && subtle.ConstantTimeCompare(gotSum[:], wantSum[:]) == 1
}
