package chatsrepo

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/chat"
	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) CreateIfNotExists(ctx context.Context, userID types.UserID) (types.ChatID, error) {
	chatID, err := r.db.Chat(ctx).Query().Where(chat.ClientIDEQ(userID)).FirstID(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return r.createChat(ctx, userID)
		}
		return types.ChatIDNil, fmt.Errorf("find existing chat: %v", err)
	}

	return chatID, nil
}

func (r *Repo) createChat(ctx context.Context, userID types.UserID) (types.ChatID, error) {
	newChat, err := r.db.Chat(ctx).Create().SetClientID(userID).Save(ctx)
	if err != nil {
		return types.ChatIDNil, fmt.Errorf("create new chat: %v", err)
	}
	return newChat.ID, nil
}
