package chatsrepo

import (
	"time"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/types"
)

type Chat struct {
	ID        types.ChatID
	ClientID  types.UserID
	CreatedAt time.Time
}

func adaptStoreChat(c *store.Chat) Chat {
	return Chat{
		ID:        c.ID,
		ClientID:  c.ClientID,
		CreatedAt: c.CreatedAt,
	}
}
