package messagesrepo

import (
	"time"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/types"
)

type Message struct {
	ID                  types.MessageID
	ChatID              types.ChatID
	AuthorID            types.UserID
	Body                string
	IsVisibleForClient  bool
	IsVisibleForManager bool
	IsBlocked           bool
	IsService           bool
	CreatedAt           time.Time
}

func adaptStoreMessage(m *store.Message) Message {
	return Message{
		ID:                  m.ID,
		ChatID:              m.ChatID,
		AuthorID:            m.AuthorID,
		Body:                m.Body,
		IsVisibleForClient:  m.IsVisibleForClient,
		IsVisibleForManager: m.IsVisibleForManager,
		IsBlocked:           m.IsBlocked,
		IsService:           m.IsService,
		CreatedAt:           m.CreatedAt,
	}
}
