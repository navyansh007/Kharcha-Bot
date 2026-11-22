package bot

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/navyansh007/kharcha-bot/internal/format"
	"github.com/navyansh007/kharcha-bot/internal/ledger"
	"github.com/navyansh007/kharcha-bot/internal/money"
	"github.com/navyansh007/kharcha-bot/internal/parser"
	"github.com/navyansh007/kharcha-bot/internal/store/db"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

const (
	groupOnlyText = "That works only in a group."

	undoTxnPrefix = "undo"
	settlePrefix  = "settle"
)

type member struct {
	id   int64
	name string
}

type unknownMemberError struct {
	label string
}

func (e *unknownMemberError) Error() string {
	return e.label + " hasn't joined yet. Ask them to send /join here."
}

func (b *Bot) join(ctx context.Context, q *db.Queries, req request, res *response) error {
	members, err := q.ListMembers(ctx, req.chat.ID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	res.reply(fmt.Sprintf("👋 %s is in. Members: %d.", format.Escape(req.user.FirstName), len(members)))
	return nil
}

func resolveMention(ctx context.Context, q *db.Queries, chatID int64, m parser.Mention) (member, error) {
	var (
		found member
		err   error
	)
	if m.UserID != 0 {
		var row db.GetMemberRow
		row, err = q.GetMember(ctx, db.GetMemberParams{ChatID: chatID, UserID: m.UserID})
		found = member{id: row.ID, name: row.FirstName}
	} else {
		var row db.FindMemberByUsernameRow
		row, err = q.FindMemberByUsername(ctx, db.FindMemberByUsernameParams{ChatID: chatID, Username: m.Username})
		found = member{id: row.ID, name: row.FirstName}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return member{}, &unknownMemberError{label: m.Label()}
	}
	if err != nil {
		return member{}, fmt.Errorf("find member: %w", err)
	}
	return found, nil
}

func splitShares(ctx context.Context, q *db.Queries, req request) ([]ledger.Share, map[int64]string, error) {
	names := map[int64]string{req.user.ID: req.user.FirstName}
	cmd := req.command

	if len(cmd.Participants) == 0 {
		members, err := q.ListMembers(ctx, req.chat.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("list members: %w", err)
		}
		ids := make([]int64, len(members))
		for i, m := range members {
			ids[i] = m.ID
			names[m.ID] = m.FirstName
		}
		return ledger.EqualShares(cmd.Amount, ids), names, nil
	}

	var shares []ledger.Share
	ids := []int64{req.user.ID}
	for _, p := range cmd.Participants {
		found, err := resolveMention(ctx, q, req.chat.ID, p.Mention)
		if err != nil {
			return nil, nil, err
		}
		names[found.id] = found.name
		ids = append(ids, found.id)
		shares = append(shares, ledger.Share{UserID: found.id, Amount: p.Share})
	}
	if cmd.Exact() {
		return shares, names, nil
	}
	return ledger.EqualShares(cmd.Amount, ids), names, nil
}

func (b *Bot) split(ctx context.Context, q *db.Queries, req request, res *response) error {
	shares, names, err := splitShares(ctx, q, req)
	if text, ok := splitFailure(err); ok {
		res.reply(text)
		return nil
	}
	if err != nil {
		return err
	}

	id, err := ledger.Split(ctx, q, ledger.SplitInput{
		ChatID:      req.chat.ID,
		PayerID:     req.user.ID,
		Total:       req.command.Amount,
		Description: req.command.Description,
		Shares:      shares,
	}, req.now)
	if text, ok := splitFailure(err); ok {
		res.reply(text)
		return nil
	}
	if err != nil {
		return err
	}

	named := make([]format.NamedShare, len(shares))
	for i, s := range shares {
		named[i] = format.NamedShare{Name: names[s.UserID], Amount: s.Amount}
	}
	res.reply(
		format.Split(req.command.Description, req.command.Amount, req.user.FirstName, named),
		row(button("Undo", fmt.Sprintf("%s:%d", undoTxnPrefix, id))),
	)
	return nil
}

func splitFailure(err error) (string, bool) {
	var (
		unknown  *unknownMemberError
		mismatch *ledger.ShareMismatchError
	)
	switch {
	case errors.As(err, &unknown):
		return format.Escape(unknown.Error()), true
	case errors.As(err, &mismatch):
		return fmt.Sprintf("Shares add up to %s, but the total is %s.", mismatch.Sum, mismatch.Total), true
	case errors.Is(err, ledger.ErrNoOtherParticipant):
		return "There is no one to split with. Ask others to send /join, or mention them.", true
	case errors.Is(err, ledger.ErrDuplicateShare):
		return "Someone is listed twice.", true
	case errors.Is(err, ledger.ErrSelfSettlement):
		return "You can't settle with yourself.", true
	default:
		return "", false
	}
}

func (b *Bot) balance(ctx context.Context, q *db.Queries, req request, res *response) error {
	balances, err := ledger.Balances(ctx, q, req.chat.ID)
	if err != nil {
		return err
	}
	transfers := ledger.Simplify(balances)

	var rows [][]telegram.InlineKeyboardButton
	for _, t := range transfers {
		label := fmt.Sprintf("Mark settled: %s → %s %s", t.From.Name, t.To.Name, t.Amount)
		data := fmt.Sprintf("%s:%d:%d:%d", settlePrefix, t.From.UserID, t.To.UserID, t.Amount)
		rows = append(rows, row(button(label, data)))
	}
	res.reply(format.Transfers("Who owes whom", transfers), rows...)
	return nil
}

func (b *Bot) settle(ctx context.Context, q *db.Queries, req request, res *response) error {
	to, err := resolveMention(ctx, q, req.chat.ID, req.command.Participants[0].Mention)
	if text, ok := splitFailure(err); ok {
		res.reply(text)
		return nil
	}
	if err != nil {
		return err
	}

	id, err := ledger.Settle(ctx, q, ledger.Settlement{
		ChatID:    req.chat.ID,
		From:      req.user.ID,
		To:        to.id,
		Amount:    req.command.Amount,
		CreatedBy: req.user.ID,
	}, req.now)
	if text, ok := splitFailure(err); ok {
		res.reply(text)
		return nil
	}
	if err != nil {
		return err
	}
	replySettled(res, req.user.FirstName, to.name, req.command.Amount, id)
	return nil
}

func replySettled(res *response, from, to string, amount money.Paise, txnID int64) {
	res.reply(
		fmt.Sprintf("🤝 %s paid %s %s.", format.Escape(from), format.Escape(to), amount),
		row(button("Undo", fmt.Sprintf("%s:%d", undoTxnPrefix, txnID))),
	)
}

func (b *Bot) undoLastTxn(ctx context.Context, q *db.Queries, req request, res *response) error {
	txn, err := ledger.UndoLastTxn(ctx, q, req.chat.ID, req.user.ID, req.now)
	if text, ok := undoFailure(err); ok {
		res.reply(text)
		return nil
	}
	if err != nil {
		return err
	}
	res.reply(fmt.Sprintf("↩️ Undid %s (%s).", format.Escape(txn.Description), txn.Total))
	return nil
}

func (b *Bot) undoTxnButton(
	ctx context.Context,
	q *db.Queries,
	cb *telegram.CallbackQuery,
	txnID int64,
	res *response,
) error {
	_, err := ledger.UndoTxn(ctx, q, cb.Message.Chat.ID, cb.From.ID, txnID, b.now())
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

func (b *Bot) settleButton(
	ctx context.Context,
	q *db.Queries,
	cb *telegram.CallbackQuery,
	from, to int64,
	amount money.Paise,
	res *response,
) error {
	if !slices.Contains([]int64{from, to}, cb.From.ID) {
		res.answer.Text = "Only the two people involved can mark this."
		return nil
	}
	chatID := cb.Message.Chat.ID

	id, err := ledger.SettleSuggested(ctx, q, ledger.Settlement{
		ChatID:    chatID,
		From:      from,
		To:        to,
		Amount:    amount,
		CreatedBy: cb.From.ID,
	}, b.now())
	if errors.Is(err, ledger.ErrStale) {
		res.answer.Text = "Balances have changed. Send /balance again."
		return nil
	}
	if err != nil {
		return err
	}

	names := make([]string, 2)
	for i, userID := range []int64{from, to} {
		m, err := q.GetUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}
		names[i] = m.FirstName
	}
	res.answer.Text = "Recorded"
	replySettled(res, names[0], names[1], amount, id)
	return nil
}
