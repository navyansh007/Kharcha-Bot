package parser

import "github.com/navyansh007/kharcha-bot/internal/money"

// Kind identifies which command a message asks for.
type Kind string

// The commands the bot understands.
const (
	KindExpense  Kind = "expense"
	KindStart    Kind = "start"
	KindHelp     Kind = "help"
	KindToday    Kind = "today"
	KindWeek     Kind = "week"
	KindMonth    Kind = "month"
	KindUndo     Kind = "undo"
	KindBudget   Kind = "budget"
	KindJoin     Kind = "join"
	KindSplit    Kind = "split"
	KindBalance  Kind = "balance"
	KindSettle   Kind = "settle"
	KindSetUPI   Kind = "setupi"
	KindSummary  Kind = "summary"
	KindExport   Kind = "export"
	KindDeleteMe Kind = "deleteme"
)

// Mention refers to a user, either by @username or by Telegram user ID.
type Mention struct {
	Username string
	UserID   int64
	Name     string
}

// Label returns the mention as the sender would recognise it.
func (m Mention) Label() string {
	if m.Username != "" {
		return "@" + m.Username
	}
	return m.Name
}

// Participant is one person in a split. Share is zero when the split is equal.
type Participant struct {
	Mention Mention
	Share   money.Paise
}

// Command is the parsed form of a message. Only the fields relevant to Kind are set.
type Command struct {
	Kind         Kind
	Amount       money.Paise
	Description  string
	Tag          string
	Category     string
	Participants []Participant
	VPA          string
	Enabled      bool
	Confirmed    bool
}

// Exact reports whether a split lists the share of each participant.
func (c Command) Exact() bool {
	return len(c.Participants) > 0 && c.Participants[0].Share > 0
}
