package getchats

import (
	"github.com/zestagio/chat-service/internal/types"
	"github.com/zestagio/chat-service/internal/validator"
)

type Request struct {
	ID        types.RequestID `validate:"required"`
	ManagerID types.UserID    `validate:"required"`
}

func (r Request) Validate() error {
	return validator.Validator.Struct(r)
}

type Response struct {
	Chats []Chat
}

type Chat struct {
	ID       types.ChatID
	ClientID types.UserID
}
