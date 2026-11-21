package ledger

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/navyansh007/kharcha-bot/internal/money"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

// Balance is a member's net position in a group. A positive amount means the
// member is owed money.
type Balance struct {
	UserID int64
	Name   string
	Amount money.Paise
}

// Transfer is a suggested payment between two members.
type Transfer struct {
	From   Balance
	To     Balance
	Amount money.Paise
}

// Balances returns the non-zero balances of a group, ordered by user ID.
func Balances(ctx context.Context, q *db.Queries, chatID int64) ([]Balance, error) {
	rows, err := q.GroupBalances(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("group balances: %w", err)
	}
	balances := make([]Balance, len(rows))
	for i, row := range rows {
		balances[i] = Balance{UserID: row.UserID, Name: row.FirstName, Amount: money.Paise(row.Balance)}
	}
	return balances, nil
}

// Simplify suggests payments that settle every balance, by repeatedly matching
// the largest creditor with the largest debtor. It produces at most n-1
// payments for n members, though not always the fewest possible.
func Simplify(balances []Balance) []Transfer {
	var creditors, debtors []Balance
	for _, b := range balances {
		switch {
		case b.Amount > 0:
			creditors = append(creditors, b)
		case b.Amount < 0:
			debtors = append(debtors, b)
		}
	}
	largestFirst := func(a, b Balance) int {
		return cmp.Or(cmp.Compare(abs(b.Amount), abs(a.Amount)), cmp.Compare(a.UserID, b.UserID))
	}

	var transfers []Transfer
	for len(creditors) > 0 && len(debtors) > 0 {
		slices.SortFunc(creditors, largestFirst)
		slices.SortFunc(debtors, largestFirst)

		amount := min(creditors[0].Amount, -debtors[0].Amount)
		transfers = append(transfers, Transfer{From: debtors[0], To: creditors[0], Amount: amount})
		creditors[0].Amount -= amount
		debtors[0].Amount += amount

		if creditors[0].Amount == 0 {
			creditors = creditors[1:]
		}
		if debtors[0].Amount == 0 {
			debtors = debtors[1:]
		}
	}
	return transfers
}

func abs(p money.Paise) money.Paise {
	if p < 0 {
		return -p
	}
	return p
}
