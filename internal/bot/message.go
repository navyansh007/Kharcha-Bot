package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/format"
	"github.com/navyansh007/kharcha-bot/internal/parser"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

const (
	chatPrivate    = "private"
	chatGroup      = "group"
	chatSuperGroup = "supergroup"
)

const helpText = `<b>Kharcha</b> keeps track of what you spend.

<b>Log an expense</b> (in a private chat)
<code>120 chai</code>
<code>450 uber to airport</code>
<code>900 shoes #shopping</code>

<b>See where it went</b>
/today, /week, /month

<b>Fix a mistake</b>
/undo removes your last entry (within 10 minutes)`

type request struct {
	chat    telegram.Chat
	user    telegram.User
	command parser.Command
	now     time.Time
}

func (r request) private() bool {
	return r.chat.Type == chatPrivate
}

func (b *Bot) handleMessage(ctx context.Context, q *db.Queries, msg *telegram.Message) (response, error) {
	res := response{chatID: msg.Chat.ID}
	if msg.From == nil || msg.From.IsBot || !supportedChat(msg.Chat.Type) {
		return res, nil
	}
	private := msg.Chat.Type == chatPrivate
	if !private && !strings.HasPrefix(msg.Text, "/") {
		return res, nil
	}
	if msg.Text == "" {
		return res, nil
	}

	command, parseErr := b.parser.Parse(msg.Text, entities(msg.Entities))
	if errors.Is(parseErr, parser.ErrOtherBot) || (!private && errors.Is(parseErr, parser.ErrUnknownCommand)) {
		return res, nil
	}
	if err := remember(ctx, q, *msg.From, msg.Chat); err != nil {
		return res, err
	}

	var usageErr *parser.UsageError
	switch {
	case errors.As(parseErr, &usageErr):
		res.command = "invalid"
		res.reply(format.Escape(usageErr.Message))
		return res, nil
	case parseErr != nil:
		res.command = "unknown"
		res.reply("I didn't understand that. Send /help to see what I can do.")
		return res, nil
	}

	res.command = string(command.Kind)
	req := request{chat: msg.Chat, user: *msg.From, command: command, now: b.now()}
	if err := b.execute(ctx, q, req, &res); err != nil {
		return res, fmt.Errorf("%s: %w", command.Kind, err)
	}
	return res, nil
}

func (b *Bot) execute(ctx context.Context, q *db.Queries, req request, res *response) error {
	switch req.command.Kind {
	case parser.KindExpense:
		return b.logExpense(ctx, q, req, res)
	case parser.KindToday, parser.KindWeek, parser.KindMonth:
		return b.totals(ctx, q, req, res)
	case parser.KindUndo:
		return b.undoLast(ctx, q, req, res)
	default:
		res.reply(helpText)
		return nil
	}
}

func supportedChat(kind string) bool {
	return kind == chatPrivate || kind == chatGroup || kind == chatSuperGroup
}

func remember(ctx context.Context, q *db.Queries, user telegram.User, chat telegram.Chat) error {
	err := q.UpsertUser(ctx, db.UpsertUserParams{
		ID:        user.ID,
		Username:  optional(user.Username),
		FirstName: user.FirstName,
	})
	if err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	err = q.UpsertChat(ctx, db.UpsertChatParams{
		ID:    chat.ID,
		Kind:  chat.Type,
		Title: optional(chat.Title),
	})
	if err != nil {
		return fmt.Errorf("save chat: %w", err)
	}
	return nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func entities(in []telegram.MessageEntity) []parser.Entity {
	out := make([]parser.Entity, len(in))
	for i, e := range in {
		out[i] = parser.Entity{Type: e.Type, Offset: e.Offset, Length: e.Length}
		if e.User != nil {
			out[i].UserID = e.User.ID
		}
	}
	return out
}
