package bot

import "github.com/navyansh007/kharcha-bot/internal/telegram"

type response struct {
	chatID    int64
	command   string
	messages  []telegram.SendMessageParams
	edits     []telegram.EditMessageTextParams
	documents []telegram.Document
	answer    *telegram.AnswerCallbackQueryParams
}

func (r *response) reply(text string, rows ...[]telegram.InlineKeyboardButton) {
	message := telegram.SendMessageParams{ChatID: r.chatID, Text: text}
	if len(rows) > 0 {
		message.ReplyMarkup = &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
	}
	r.messages = append(r.messages, message)
}

func button(text, data string) telegram.InlineKeyboardButton {
	return telegram.InlineKeyboardButton{Text: text, CallbackData: data}
}

func row(buttons ...telegram.InlineKeyboardButton) []telegram.InlineKeyboardButton {
	return buttons
}
