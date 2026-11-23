package cron

import (
	"context"
	"errors"
	"net/http"

	"github.com/navyansh007/kharcha-bot/internal/outbox"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

// TelegramSender delivers outbox messages through the Bot API.
type TelegramSender struct {
	Client telegram.Client
}

// Send sends text to a chat and classifies any failure for the outbox.
func (s TelegramSender) Send(ctx context.Context, chatID int64, text string) error {
	err := s.Client.SendMessage(ctx, telegram.SendMessageParams{ChatID: chatID, Text: text})
	var apiErr *telegram.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	return &outbox.SendError{
		Err:        err,
		RetryAfter: apiErr.RetryAfter,
		Blocked:    apiErr.Code == http.StatusForbidden,
		Permanent:  apiErr.Code == http.StatusBadRequest,
	}
}
