package bot

import (
	"context"

	"github.com/navyansh007/kharcha-bot/internal/ratelimit"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

const slowDownText = "You're sending too much too quickly, so that one was not saved. Try again in a little while."

func limitedRequest(update telegram.Update) (ratelimit.Request, bool) {
	var (
		user telegram.User
		chat telegram.Chat
	)
	switch {
	case update.Message != nil && update.Message.From != nil && update.Message.LeftChatMember == nil:
		user, chat = *update.Message.From, update.Message.Chat
	case update.CallbackQuery != nil && update.CallbackQuery.Message != nil:
		user, chat = update.CallbackQuery.From, update.CallbackQuery.Message.Chat
	case update.CallbackQuery != nil:
		user = update.CallbackQuery.From
	case update.MyChatMember != nil:
		user, chat = update.MyChatMember.From, update.MyChatMember.Chat
	default:
		return ratelimit.Request{}, false
	}
	return ratelimit.Request{
		UserID:  user.ID,
		ChatID:  chat.ID,
		IsGroup: chat.Type == chatGroup || chat.Type == chatSuperGroup,
	}, true
}

func (b *Bot) refuse(ctx context.Context, update telegram.Update, request ratelimit.Request) {
	if !b.limits.ShouldNotify(request.UserID, b.now()) {
		return
	}
	result := response{chatID: request.ChatID}
	if cb := update.CallbackQuery; cb != nil {
		result.answer = &telegram.AnswerCallbackQueryParams{CallbackQueryID: cb.ID, Text: slowDownText}
	} else if update.Message != nil {
		result.reply(slowDownText)
	}
	b.deliver(context.WithoutCancel(ctx), result)
}
