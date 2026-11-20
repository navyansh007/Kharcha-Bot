package bot

import (
	"context"
	"strconv"
	"strings"

	"github.com/navyansh007/kharcha-bot/internal/format"
	"github.com/navyansh007/kharcha-bot/internal/ledger"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

func (b *Bot) handleCallback(ctx context.Context, q *db.Queries, cb *telegram.CallbackQuery) (response, error) {
	res := response{answer: &telegram.AnswerCallbackQueryParams{CallbackQueryID: cb.ID}}
	if cb.Message == nil || !supportedChat(cb.Message.Chat.Type) {
		res.answer.Text = "This button no longer works."
		return res, nil
	}
	res.chatID = cb.Message.Chat.ID
	if err := remember(ctx, q, cb.From, cb.Message.Chat); err != nil {
		return res, err
	}

	action, args, _ := strings.Cut(cb.Data, ":")
	res.command = "callback_" + action
	ids, ok := parseIDs(args)
	switch {
	case ok && action == undoExpensePrefix && len(ids) == 1:
		return res, b.undoExpenseButton(ctx, q, cb, ids[0], &res)
	default:
		res.command = "callback_unknown"
		res.answer.Text = "This button no longer works."
		return res, nil
	}
}

func (b *Bot) undoExpenseButton(
	ctx context.Context,
	q *db.Queries,
	cb *telegram.CallbackQuery,
	id int64,
	res *response,
) error {
	_, err := ledger.UndoExpense(ctx, q, cb.From.ID, id, b.now())
	if text, ok := undoFailure(err); ok {
		res.answer.Text = text
		return nil
	}
	if err != nil {
		return err
	}
	res.answer.Text = "Undone"
	strikeThrough(cb.Message, res)
	return nil
}

func strikeThrough(msg *telegram.Message, res *response) {
	res.edits = append(res.edits, telegram.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.MessageID,
		Text:      "<s>" + format.Escape(msg.Text) + "</s>",
	})
}

func parseIDs(s string) ([]int64, bool) {
	var ids []int64
	for part := range strings.SplitSeq(s, ":") {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}
