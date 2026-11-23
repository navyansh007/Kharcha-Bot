package bot

import (
	"context"
	"fmt"

	"github.com/navyansh007/kharcha-bot/internal/store/db"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

func (b *Bot) handleMembership(
	ctx context.Context,
	q *db.Queries,
	change *telegram.ChatMemberUpdated,
) (response, error) {
	res := response{chatID: change.Chat.ID, command: "membership"}
	if !supportedChat(change.Chat.Type) {
		return res, nil
	}
	if err := remember(ctx, q, change.From, change.Chat); err != nil {
		return res, err
	}
	if status := change.NewChatMember.Status; status != "kicked" && status != "left" {
		return res, nil
	}

	var err error
	if change.Chat.Type == chatPrivate {
		err = q.MarkUserBlocked(ctx, change.From.ID)
	} else {
		err = q.MarkChatRemoved(ctx, change.Chat.ID)
	}
	if err != nil {
		return res, fmt.Errorf("record removal: %w", err)
	}
	return res, nil
}
