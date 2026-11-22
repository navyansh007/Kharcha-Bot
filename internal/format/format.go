// Package format builds the HTML text of the bot's messages.
package format

import (
	"fmt"
	"html"
	"strings"

	"github.com/navyansh007/kharcha-bot/internal/ledger"
	"github.com/navyansh007/kharcha-bot/internal/money"
)

// Escape makes user supplied text safe to embed in an HTML message.
func Escape(s string) string {
	return html.EscapeString(s)
}

// Totals renders spending by category under a title.
func Totals(title string, totals []ledger.CategoryTotal) string {
	if len(totals) == 0 {
		return fmt.Sprintf("<b>%s</b>\nNothing logged yet.", Escape(title))
	}
	var b strings.Builder
	var sum money.Paise
	fmt.Fprintf(&b, "<b>%s</b>\n", Escape(title))
	for _, t := range totals {
		fmt.Fprintf(&b, "%s: %s\n", Escape(t.Category), t.Total)
		sum += t.Total
	}
	fmt.Fprintf(&b, "\n<b>Total: %s</b>", sum)
	return b.String()
}

// TransferLine renders one suggested payment.
func TransferLine(t ledger.Transfer) string {
	return fmt.Sprintf("%s → %s: %s", Escape(t.From.Name), Escape(t.To.Name), t.Amount)
}

// Transfers renders the payments that would settle a group.
func Transfers(title string, transfers []ledger.Transfer) string {
	if len(transfers) == 0 {
		return "All settled up. 🎉"
	}
	lines := []string{fmt.Sprintf("<b>%s</b>", Escape(title))}
	for _, t := range transfers {
		lines = append(lines, TransferLine(t))
	}
	return strings.Join(lines, "\n")
}

// NamedShare is a participant's part of a split.
type NamedShare struct {
	Name   string
	Amount money.Paise
}

// Split renders the confirmation of a shared expense.
func Split(description string, total money.Paise, payer string, shares []NamedShare) string {
	header := fmt.Sprintf("🧾 <b>%s</b>, %s split %d %s", Escape(description), total, len(shares), plural(len(shares), "way"))
	even := true
	for _, s := range shares {
		even = even && s.Amount == shares[0].Amount
	}
	if even {
		header += fmt.Sprintf(" (%s each)", shares[0].Amount)
	}

	lines := []string{header + ". Paid by " + Escape(payer) + "."}
	for _, s := range shares {
		lines = append(lines, fmt.Sprintf("• %s: %s", Escape(s.Name), s.Amount))
	}
	return strings.Join(lines, "\n")
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
