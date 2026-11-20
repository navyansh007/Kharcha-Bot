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
