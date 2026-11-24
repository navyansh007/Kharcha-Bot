// Package export writes a user's records as CSV.
package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

var header = []string{"date", "type", "group", "category", "description", "amount_inr"}

// CSV returns the user's personal expenses and group entries. It reports false
// if the user has none.
func CSV(ctx context.Context, q *db.Queries, userID int64, loc *time.Location) ([]byte, bool, error) {
	expenses, err := q.ExportPersonalExpenses(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("export expenses: %w", err)
	}
	entries, err := q.ExportGroupEntries(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("export group entries: %w", err)
	}
	if len(expenses)+len(entries) == 0 {
		return nil, false, nil
	}

	date := func(t time.Time) string { return t.In(loc).Format("2006-01-02 15:04") }
	records := [][]string{header}
	for _, e := range expenses {
		records = append(records, []string{
			date(e.SpentAt), "personal", "", safeCell(e.Category), safeCell(e.Description), e.Amount.Rupees(),
		})
	}
	for _, e := range entries {
		title := ""
		if e.Title != nil {
			title = *e.Title
		}
		records = append(records, []string{
			date(e.CreatedAt), e.Kind, safeCell(title), "", safeCell(e.Description), e.Amount.Rupees(),
		})
	}

	var buf bytes.Buffer
	if err := csv.NewWriter(&buf).WriteAll(records); err != nil {
		return nil, false, fmt.Errorf("write csv: %w", err)
	}
	return buf.Bytes(), true, nil
}

func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
