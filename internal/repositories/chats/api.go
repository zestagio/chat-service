package chatsrepo

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/store/chat"
	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) CreateIfNotExists(ctx context.Context, userID types.UserID) (types.ChatID, error) {
	chatID, err := r.db.Chat(ctx).Create().
		SetClientID(userID).
		OnConflictColumns(chat.FieldClientID).Ignore().
		// More performant way:
		//	OnConflict(
		//		sql.ConflictColumns(chat.FieldClientID),
		//		sql.ResolveWith(func(set *sql.UpdateSet) {
		//			set.SetIgnore(chat.FieldClientID)
		//		}),
		//	).
		ID(ctx)
	if err != nil {
		return types.ChatIDNil, fmt.Errorf("create new chat: %v", err)
	}

	return chatID, nil
}
