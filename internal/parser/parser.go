// Package parser turns message text into commands. It performs no I/O.
package parser

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/navyansh007/kharcha-bot/internal/money"
)

// Telegram entity types the parser looks at.
const (
	EntityMention     = "mention"
	EntityTextMention = "text_mention"
)

const maxDescriptionRunes = 100

var (
	wordPattern = regexp.MustCompile(`^[\p{L}\p{N}_-]{1,24}$`)
	vpaPattern  = regexp.MustCompile(`^[a-zA-Z0-9._-]{2,256}@[a-zA-Z][a-zA-Z0-9.-]{1,63}$`)
)

// Errors returned by Parse, besides UsageError.
var (
	ErrNotUnderstood  = errors.New("message not understood")
	ErrUnknownCommand = errors.New("unknown command")
	ErrOtherBot       = errors.New("command is addressed to another bot")
)

// UsageError describes a known command that was written incorrectly.
type UsageError struct {
	Message string
}

func (e *UsageError) Error() string {
	return e.Message
}

func usage(format string, args ...any) error {
	return &UsageError{Message: fmt.Sprintf(format, args...)}
}

// Entity is a Telegram message entity. Offset and Length count UTF-16 code units.
type Entity struct {
	Type   string
	Offset int
	Length int
	UserID int64
}

// Parser holds the limits applied while parsing.
type Parser struct {
	MaxAmount   money.Paise
	BotUsername string
}

// Parse converts a message and its entities into a Command.
func (p Parser) Parse(text string, entities []Entity) (Command, error) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return p.expense(trimmed)
	}

	name, rest := cutField(trimmed)
	name, addressee, addressed := strings.Cut(strings.TrimPrefix(name, "/"), "@")
	if addressed && p.BotUsername != "" && !strings.EqualFold(addressee, p.BotUsername) {
		return Command{}, ErrOtherBot
	}

	args := strings.Fields(rest)
	switch kind := Kind(strings.ToLower(name)); kind {
	case KindStart, KindHelp, KindToday, KindWeek, KindMonth, KindUndo, KindJoin, KindBalance, KindExport:
		return Command{Kind: kind}, nil
	case KindBudget:
		return p.budget(args)
	case KindSplit:
		return p.split(text, entities)
	case KindSettle:
		return p.settle(text, entities)
	case KindSetUPI:
		return setUPI(args)
	case KindSummary:
		return summary(args)
	case KindDeleteMe:
		confirmed := len(args) == 1 && strings.EqualFold(args[0], "confirm")
		return Command{Kind: kind, Confirmed: confirmed}, nil
	default:
		return Command{}, ErrUnknownCommand
	}
}

func (p Parser) amount(s string) (money.Paise, error) {
	amount, err := money.Parse(s)
	if err != nil {
		return 0, usage("%q is not a valid amount", s)
	}
	if amount <= 0 {
		return 0, usage("The amount must be more than zero")
	}
	if amount > p.MaxAmount {
		return 0, usage("The amount can't be more than %s", p.MaxAmount)
	}
	return amount, nil
}

func (p Parser) expense(text string) (Command, error) {
	fields := strings.Fields(text)
	tag := ""
	if n := len(fields); n > 0 && strings.HasPrefix(fields[n-1], "#") {
		tag = strings.ToLower(strings.TrimPrefix(fields[n-1], "#"))
		if !wordPattern.MatchString(tag) {
			return Command{}, ErrNotUnderstood
		}
		fields = fields[:n-1]
	}
	if len(fields) < 2 {
		return Command{}, ErrNotUnderstood
	}

	var amountField string
	var description []string
	switch {
	case looksLikeAmount(fields[0]):
		amountField, description = fields[0], fields[1:]
	case looksLikeAmount(fields[len(fields)-1]):
		amountField, description = fields[len(fields)-1], fields[:len(fields)-1]
	default:
		return Command{}, ErrNotUnderstood
	}

	amount, err := p.amount(amountField)
	if err != nil {
		return Command{}, err
	}
	desc, err := describe(description)
	if err != nil {
		return Command{}, err
	}
	return Command{Kind: KindExpense, Amount: amount, Description: desc, Tag: tag}, nil
}

func looksLikeAmount(s string) bool {
	_, err := money.Parse(s)
	return err == nil
}

func describe(words []string) (string, error) {
	desc := strings.ToValidUTF8(strings.Join(words, " "), "")
	desc = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, desc)
	if desc == "" {
		return "", usage("Add a short description")
	}
	if len([]rune(desc)) > maxDescriptionRunes {
		return "", usage("Keep the description under %d characters", maxDescriptionRunes)
	}
	return desc, nil
}

func (p Parser) budget(args []string) (Command, error) {
	if len(args) != 2 {
		return Command{}, usage("Usage: /budget food 3000")
	}
	name := strings.ToLower(strings.TrimPrefix(args[0], "#"))
	if !wordPattern.MatchString(name) {
		return Command{}, usage("%q is not a valid category", args[0])
	}
	amount, err := p.amount(args[1])
	if err != nil {
		return Command{}, err
	}
	return Command{Kind: KindBudget, Category: name, Amount: amount}, nil
}

const splitUsage = "Usage: /split 900 dinner, optionally followed by @a @b or @a:300 @b:600"

func (p Parser) split(text string, entities []Entity) (Command, error) {
	head, mentions := cutAtMentions(text, entities)
	_, head = cutField(strings.TrimSpace(head))
	fields := strings.Fields(head)
	if len(fields) < 2 {
		return Command{}, usage(splitUsage)
	}
	total, err := p.amount(fields[0])
	if err != nil {
		return Command{}, err
	}
	desc, err := describe(fields[1:])
	if err != nil {
		return Command{}, err
	}

	cmd := Command{Kind: KindSplit, Amount: total, Description: desc}
	for _, m := range mentions {
		participant := Participant{Mention: m.mention}
		switch trailing := strings.TrimSpace(m.trailing); {
		case trailing == "":
		case strings.HasPrefix(trailing, ":"):
			share, err := p.amount(strings.TrimSpace(trailing[1:]))
			if err != nil {
				return Command{}, err
			}
			participant.Share = share
		default:
			return Command{}, usage(splitUsage)
		}
		cmd.Participants = append(cmd.Participants, participant)
	}

	withShare := 0
	for _, participant := range cmd.Participants {
		if participant.Share > 0 {
			withShare++
		}
	}
	if withShare != 0 && withShare != len(cmd.Participants) {
		return Command{}, usage("Give a share for everyone, or for no one")
	}
	return cmd, nil
}

const settleUsage = "Usage: /settle @name 300"

func (p Parser) settle(text string, entities []Entity) (Command, error) {
	head, mentions := cutAtMentions(text, entities)
	if len(strings.Fields(head)) != 1 || len(mentions) != 1 {
		return Command{}, usage(settleUsage)
	}
	amount, err := p.amount(strings.TrimSpace(mentions[0].trailing))
	if err != nil {
		return Command{}, err
	}
	return Command{
		Kind:         KindSettle,
		Amount:       amount,
		Participants: []Participant{{Mention: mentions[0].mention}},
	}, nil
}

type mentionSpan struct {
	mention  Mention
	trailing string
}

func cutAtMentions(text string, entities []Entity) (string, []mentionSpan) {
	units := utf16.Encode([]rune(text))
	decode := func(from, to int) string {
		return string(utf16.Decode(units[from:to]))
	}

	var found []Entity
	for _, e := range entities {
		if e.Type != EntityMention && e.Type != EntityTextMention {
			continue
		}
		if e.Offset < 0 || e.Length <= 0 || e.Offset > len(units) || e.Length > len(units)-e.Offset {
			continue
		}
		found = append(found, e)
	}
	slices.SortFunc(found, func(a, b Entity) int { return a.Offset - b.Offset })

	if len(found) == 0 {
		return text, nil
	}

	var spans []mentionSpan
	for i, e := range found {
		end := e.Offset + e.Length
		next := len(units)
		if i+1 < len(found) {
			next = max(found[i+1].Offset, end)
		}
		label := decode(e.Offset, end)
		m := Mention{Name: label}
		if e.Type == EntityTextMention {
			m.UserID = e.UserID
		} else {
			m = Mention{Username: strings.TrimPrefix(label, "@")}
		}
		spans = append(spans, mentionSpan{mention: m, trailing: decode(end, max(next, end))})
	}
	return decode(0, found[0].Offset), spans
}

func setUPI(args []string) (Command, error) {
	if len(args) != 1 || !vpaPattern.MatchString(args[0]) {
		return Command{}, usage("Usage: /setupi name@okaxis")
	}
	return Command{Kind: KindSetUPI, VPA: args[0]}, nil
}

func summary(args []string) (Command, error) {
	if len(args) == 1 {
		switch strings.ToLower(args[0]) {
		case "on":
			return Command{Kind: KindSummary, Enabled: true}, nil
		case "off":
			return Command{Kind: KindSummary}, nil
		}
	}
	return Command{}, usage("Usage: /summary on or /summary off")
}

func cutField(s string) (field, rest string) {
	i := strings.IndexFunc(s, unicode.IsSpace)
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i:]
}
