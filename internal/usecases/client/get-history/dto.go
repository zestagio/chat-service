package gethistory

import (
	"time"

	"github.com/zestagio/chat-service/internal/types"
	"github.com/zestagio/chat-service/internal/validator"
)

type Request struct {
	ID       types.RequestID `validate:"required"`
	ClientID types.UserID    `validate:"required"`
	PageSize int             `validate:"required_without=Cursor,excluded_with=Cursor,omitempty,gte=10,lte=100"`
	Cursor   string          `validate:"required_without=PageSize,excluded_with=PageSize,omitempty,base64url"`
}

func (r Request) Validate() error {
	return validator.Validator.Struct(r)
}

type Response struct {
	Messages   []Message
	NextCursor string
}

type Message struct {
	ID                  types.MessageID
	AuthorID            types.UserID
	Body                string
	IsVisibleForManager bool
	IsBlocked           bool
	IsReceived          bool
	IsService           bool
	CreatedAt           time.Time
}
