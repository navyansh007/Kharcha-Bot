// Package ledger holds the rules for expenses, splits, balances and settlements.
package ledger

import (
	"errors"
	"time"
)

// UndoWindow is how long after creation an entry can still be undone.
const UndoWindow = 10 * time.Minute

// Errors returned when an entry cannot be undone.
var (
	ErrNotFound      = errors.New("entry not found")
	ErrNotOwner      = errors.New("entry belongs to someone else")
	ErrTooLate       = errors.New("undo window has passed")
	ErrAlreadyUndone = errors.New("entry already undone")
)
