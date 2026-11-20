package telegram

import (
	"context"
	"sync"
	"time"
)

// Fake is an in-memory Client for tests. It is safe for concurrent use.
type Fake struct {
	mu        sync.Mutex
	Sent      []SendMessageParams
	Edited    []EditMessageTextParams
	Answered  []AnswerCallbackQueryParams
	Documents []Document
	Updates   []Update
	SendErr   func(SendMessageParams) error
}

var _ Client = (*Fake)(nil)

// GetMe returns a fixed bot account.
func (f *Fake) GetMe(context.Context) (User, error) {
	return User{ID: 1, IsBot: true, FirstName: "Kharcha", Username: "KharchaBot"}, nil
}

// GetUpdates returns the queued updates once, then blocks until ctx is done.
func (f *Fake) GetUpdates(ctx context.Context, _ int64, _ time.Duration) ([]Update, error) {
	f.mu.Lock()
	updates := f.Updates
	f.Updates = nil
	f.mu.Unlock()
	if len(updates) > 0 {
		return updates, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

// SendMessage records the message.
func (f *Fake) SendMessage(_ context.Context, params SendMessageParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.SendErr != nil {
		if err := f.SendErr(params); err != nil {
			return err
		}
	}
	f.Sent = append(f.Sent, params)
	return nil
}

// EditMessageText records the edit.
func (f *Fake) EditMessageText(_ context.Context, params EditMessageTextParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Edited = append(f.Edited, params)
	return nil
}

// AnswerCallbackQuery records the answer.
func (f *Fake) AnswerCallbackQuery(_ context.Context, params AnswerCallbackQueryParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Answered = append(f.Answered, params)
	return nil
}

// SendDocument records the document.
func (f *Fake) SendDocument(_ context.Context, doc Document) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Documents = append(f.Documents, doc)
	return nil
}

// Reset forgets everything recorded so far.
func (f *Fake) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Sent, f.Edited, f.Answered, f.Documents = nil, nil, nil, nil
}
