package telegram

// Update is one incoming event from Telegram.
type Update struct {
	UpdateID      int64              `json:"update_id"`
	Message       *Message           `json:"message,omitempty"`
	CallbackQuery *CallbackQuery     `json:"callback_query,omitempty"`
	MyChatMember  *ChatMemberUpdated `json:"my_chat_member,omitempty"`
}

// User is a Telegram user or bot.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

// Chat is a private chat, group, supergroup or channel.
type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

// MessageEntity marks a special part of a message, such as a mention.
type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	User   *User  `json:"user,omitempty"`
}

// Message is a message in a chat.
type Message struct {
	MessageID      int64           `json:"message_id"`
	Date           int64           `json:"date,omitempty"`
	From           *User           `json:"from,omitempty"`
	Chat           Chat            `json:"chat"`
	Text           string          `json:"text,omitempty"`
	Entities       []MessageEntity `json:"entities,omitempty"`
	LeftChatMember *User           `json:"left_chat_member,omitempty"`
}

// CallbackQuery is a press on an inline keyboard button.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// ChatMember describes a user's standing in a chat.
type ChatMember struct {
	Status string `json:"status"`
}

// ChatMemberUpdated reports a change to the bot's own membership of a chat.
type ChatMemberUpdated struct {
	Chat          Chat       `json:"chat"`
	From          User       `json:"from"`
	NewChatMember ChatMember `json:"new_chat_member"`
}

// InlineKeyboardButton is a button shown under a message.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboardMarkup is a grid of inline buttons.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// SendMessageParams are the arguments of sendMessage. Text is always HTML.
type SendMessageParams struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// EditMessageTextParams are the arguments of editMessageText. Text is always HTML.
type EditMessageTextParams struct {
	ChatID      int64                 `json:"chat_id"`
	MessageID   int64                 `json:"message_id"`
	Text        string                `json:"text"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// AnswerCallbackQueryParams are the arguments of answerCallbackQuery.
type AnswerCallbackQueryParams struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
}

// Document is a file to upload with sendDocument.
type Document struct {
	ChatID  int64
	Name    string
	Content []byte
}
