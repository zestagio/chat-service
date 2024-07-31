package messagesrepo

import (
	"time"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/types"
)

type Message struct {
	ID                  types.MessageID
	ChatID              types.ChatID
	ProblemID           types.ProblemID
	AuthorID            types.UserID
	ManagerID           types.UserID
	Body                string
	CreatedAt           time.Time
	IsVisibleForClient  bool
	IsVisibleForManager bool
	IsBlocked           bool
	IsService           bool
	InitialRequestID    types.RequestID
}

func adaptStoreMessage(m *store.Message) Message {
	managerID := types.UserIDNil
	if p := m.Edges.Problem; p != nil {
		managerID = p.ManagerID
	}

	return Message{
		ID:                  m.ID,
		ChatID:              m.ChatID,
		ProblemID:           m.ProblemID,
		AuthorID:            m.AuthorID,
		ManagerID:           managerID,
		Body:                m.Body,
		CreatedAt:           m.CreatedAt,
		IsVisibleForClient:  m.IsVisibleForClient,
		IsVisibleForManager: m.IsVisibleForManager,
		IsBlocked:           m.IsBlocked,
		IsService:           m.IsService,
		InitialRequestID:    m.InitialRequestID,
	}
}
