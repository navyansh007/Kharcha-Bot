package ledger

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/navyansh007/kharcha-bot/internal/money"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

// The kinds of group transaction.
const (
	KindExpense    = "expense"
	KindSettlement = "settlement"
	KindReversal   = "reversal"
)

// Errors returned when a split or settlement is rejected.
var (
	ErrNoOtherParticipant = errors.New("a split needs someone besides the payer")
	ErrDuplicateShare     = errors.New("a participant is listed twice")
	ErrSelfSettlement     = errors.New("cannot settle with yourself")
	ErrStale              = errors.New("balances have changed")
)

var errUnbalanced = errors.New("entries do not sum to zero")

// ShareMismatchError reports exact shares that do not add up to the total.
type ShareMismatchError struct {
	Sum   money.Paise
	Total money.Paise
}

func (e *ShareMismatchError) Error() string {
	return fmt.Sprintf("shares add up to %s, but the total is %s", e.Sum, e.Total)
}

// Share is the part of an expense one user is responsible for.
type Share struct {
	UserID int64
	Amount money.Paise
}

// Entry changes a user's balance. A positive amount means others owe the user.
type Entry struct {
	UserID int64
	Amount money.Paise
}

// SplitInput describes a shared expense.
type SplitInput struct {
	ChatID      int64
	PayerID     int64
	Total       money.Paise
	Description string
	Shares      []Share
}

// EqualShares divides total between the users. Duplicate IDs count once, and
// any leftover paise go to the lowest user IDs.
func EqualShares(total money.Paise, userIDs []int64) []Share {
	ids := slices.Clone(userIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)

	amounts := money.SplitEqual(total, len(ids))
	shares := make([]Share, len(ids))
	for i, id := range ids {
		shares[i] = Share{UserID: id, Amount: amounts[i]}
	}
	return shares
}

func splitEntries(in SplitInput) ([]Entry, error) {
	net := map[int64]money.Paise{in.PayerID: in.Total}
	seen := map[int64]bool{}
	var sum money.Paise
	others := false
	for _, share := range in.Shares {
		if seen[share.UserID] {
			return nil, ErrDuplicateShare
		}
		seen[share.UserID] = true
		sum += share.Amount
		net[share.UserID] -= share.Amount
		others = others || (share.UserID != in.PayerID && share.Amount > 0)
	}
	if sum != in.Total {
		return nil, &ShareMismatchError{Sum: sum, Total: in.Total}
	}
	if !others {
		return nil, ErrNoOtherParticipant
	}

	entries := make([]Entry, 0, len(net))
	for id, amount := range net {
		if amount != 0 {
			entries = append(entries, Entry{UserID: id, Amount: amount})
		}
	}
	slices.SortFunc(entries, func(a, b Entry) int { return int(a.UserID - b.UserID) })
	return entries, nil
}

// Split records a shared expense and returns the transaction ID.
func Split(ctx context.Context, q *db.Queries, in SplitInput, now time.Time) (int64, error) {
	entries, err := splitEntries(in)
	if err != nil {
		return 0, err
	}
	return insertTxn(ctx, q, db.InsertGroupTxnParams{
		ChatID:      in.ChatID,
		Kind:        KindExpense,
		CreatedBy:   in.PayerID,
		Description: in.Description,
		Total:       in.Total,
		CreatedAt:   now,
	}, entries)
}

// Settlement records that From paid To outside the bot.
type Settlement struct {
	ChatID    int64
	From      int64
	To        int64
	Amount    money.Paise
	CreatedBy int64
}

// Settle records a settlement and returns the transaction ID.
func Settle(ctx context.Context, q *db.Queries, s Settlement, now time.Time) (int64, error) {
	if s.From == s.To {
		return 0, ErrSelfSettlement
	}
	return insertTxn(ctx, q, db.InsertGroupTxnParams{
		ChatID:      s.ChatID,
		Kind:        KindSettlement,
		CreatedBy:   s.CreatedBy,
		Description: "settlement",
		Total:       s.Amount,
		CreatedAt:   now,
	}, []Entry{{UserID: s.From, Amount: s.Amount}, {UserID: s.To, Amount: -s.Amount}})
}

// SettleSuggested records a settlement only if it is still one of the suggested
// transfers, so that pressing the same button twice has no further effect.
func SettleSuggested(ctx context.Context, q *db.Queries, s Settlement, now time.Time) (int64, error) {
	if err := q.LockChat(ctx, s.ChatID); err != nil {
		return 0, fmt.Errorf("lock chat: %w", err)
	}
	balances, err := Balances(ctx, q, s.ChatID)
	if err != nil {
		return 0, err
	}
	current := slices.ContainsFunc(Simplify(balances), func(t Transfer) bool {
		return t.From.UserID == s.From && t.To.UserID == s.To && t.Amount == s.Amount
	})
	if !current {
		return 0, ErrStale
	}
	return Settle(ctx, q, s, now)
}

func insertTxn(ctx context.Context, q *db.Queries, txn db.InsertGroupTxnParams, entries []Entry) (int64, error) {
	if err := checkBalanced(entries); err != nil {
		return 0, err
	}
	id, err := q.InsertGroupTxn(ctx, txn)
	if err != nil {
		return 0, fmt.Errorf("insert txn: %w", err)
	}
	return id, insertEntries(ctx, q, id, entries)
}

func checkBalanced(entries []Entry) error {
	var sum money.Paise
	for _, e := range entries {
		sum += e.Amount
	}
	if sum != 0 {
		return errUnbalanced
	}
	return nil
}

func insertEntries(ctx context.Context, q *db.Queries, txnID int64, entries []Entry) error {
	for _, e := range entries {
		err := q.InsertGroupEntry(ctx, db.InsertGroupEntryParams{TxnID: txnID, UserID: e.UserID, Amount: e.Amount})
		if err != nil {
			return fmt.Errorf("insert entry: %w", err)
		}
	}
	return nil
}

// UndoTxn reverses a group transaction created by userID.
func UndoTxn(ctx context.Context, q *db.Queries, chatID, userID, txnID int64, now time.Time) (db.GroupTxn, error) {
	txn, err := q.GetGroupTxn(ctx, db.GetGroupTxnParams{ID: txnID, ChatID: chatID})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GroupTxn{}, ErrNotFound
	}
	if err != nil {
		return db.GroupTxn{}, fmt.Errorf("get txn: %w", err)
	}
	return txn, reverse(ctx, q, txn, userID, now)
}

// UndoLastTxn reverses the most recent transaction userID created in the chat.
func UndoLastTxn(ctx context.Context, q *db.Queries, chatID, userID int64, now time.Time) (db.GroupTxn, error) {
	txn, err := q.LastUndoableTxn(ctx, db.LastUndoableTxnParams{ChatID: chatID, CreatedBy: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GroupTxn{}, ErrNotFound
	}
	if err != nil {
		return db.GroupTxn{}, fmt.Errorf("get last txn: %w", err)
	}
	return txn, reverse(ctx, q, txn, userID, now)
}

func reverse(ctx context.Context, q *db.Queries, txn db.GroupTxn, userID int64, now time.Time) error {
	switch {
	case txn.Kind == KindReversal:
		return ErrNotFound
	case txn.CreatedBy != userID:
		return ErrNotOwner
	case now.Sub(txn.CreatedAt) > UndoWindow:
		return ErrTooLate
	}

	rows, err := q.ListGroupEntries(ctx, txn.ID)
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}
	id, err := q.InsertReversalTxn(ctx, db.InsertReversalTxnParams{
		ChatID:      txn.ChatID,
		CreatedBy:   userID,
		Description: txn.Description,
		Total:       txn.Total,
		ReversesID:  &txn.ID,
		CreatedAt:   now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAlreadyUndone
	}
	if err != nil {
		return fmt.Errorf("insert reversal: %w", err)
	}

	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entries[i] = Entry{UserID: row.UserID, Amount: -row.Amount}
	}
	return insertEntries(ctx, q, id, entries)
}
