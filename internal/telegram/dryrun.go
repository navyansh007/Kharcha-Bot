package telegram

import (
	"context"
	"log/slog"
	"time"
)

// DryRun is a Client that sends nothing. It is used for load tests.
type DryRun struct {
	Logger *slog.Logger
}

var _ Client = DryRun{}

// GetMe returns a placeholder bot account.
func (d DryRun) GetMe(context.Context) (User, error) {
	return User{IsBot: true, FirstName: "DryRun"}, nil
}

// GetUpdates blocks until ctx is done.
func (d DryRun) GetUpdates(ctx context.Context, _ int64, _ time.Duration) ([]Update, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// SendMessage logs the destination chat.
func (d DryRun) SendMessage(_ context.Context, params SendMessageParams) error {
	d.Logger.Debug("dry run: sendMessage", "chat_id", params.ChatID)
	return nil
}

// EditMessageText logs the destination chat.
func (d DryRun) EditMessageText(_ context.Context, params EditMessageTextParams) error {
	d.Logger.Debug("dry run: editMessageText", "chat_id", params.ChatID)
	return nil
}

// AnswerCallbackQuery does nothing.
func (d DryRun) AnswerCallbackQuery(context.Context, AnswerCallbackQueryParams) error {
	return nil
}

// SendDocument logs the destination chat.
func (d DryRun) SendDocument(_ context.Context, doc Document) error {
	d.Logger.Debug("dry run: sendDocument", "chat_id", doc.ChatID)
	return nil
}
