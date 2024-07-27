package eventstream

import (
	"time"

	"github.com/zestagio/chat-service/internal/types"
	"github.com/zestagio/chat-service/internal/validator"
)

//go:generate gonstructor --output=events.gen.go --type=NewMessageEvent --type=MessageSentEvent --type=MessageBlockedEvent --type=NewChatEvent --type=ChatClosedEvent

type Event interface {
	eventMarker()
	Validate() error
}

type event struct{}         //
func (*event) eventMarker() {}

// NewMessageEvent is a signal about the appearance of a new message in the chat.
type NewMessageEvent struct {
	event       `gonstructor:"-"`
	EventID     types.EventID   `validate:"required"`
	RequestID   types.RequestID `validate:"required"`
	ChatID      types.ChatID    `validate:"required"`
	MessageID   types.MessageID `validate:"required"`
	AuthorID    types.UserID    // Zero if IsService == true.
	CreatedAt   time.Time       `validate:"required"`
	MessageBody string          `validate:"required,max=3000"`
	IsService   bool
}

func (e NewMessageEvent) Validate() error { return validator.Validator.Struct(e) }

// MessageSentEvent indicates that the message was checked by AFC
// and was sent to the manager. Two gray ticks.
type MessageSentEvent struct {
	event     `gonstructor:"-"`
	EventID   types.EventID   `validate:"required"`
	RequestID types.RequestID `validate:"required"`
	MessageID types.MessageID `validate:"required"`
}

func (e MessageSentEvent) Validate() error { return validator.Validator.Struct(e) }

// MessageBlockedEvent indicates that AFC recognized the message as suspicious.
// The manager will not receive it, but the client will be warned.
type MessageBlockedEvent struct {
	event     `gonstructor:"-"`
	EventID   types.EventID   `validate:"required"`
	RequestID types.RequestID `validate:"required"`
	MessageID types.MessageID `validate:"required"`
}

func (e MessageBlockedEvent) Validate() error { return validator.Validator.Struct(e) }

// NewChatEvent is a signal about the appearance of a new chat.
type NewChatEvent struct {
	event              `gonstructor:"-"`
	EventID            types.EventID   `validate:"required"`
	RequestID          types.RequestID `validate:"required"`
	ChatID             types.ChatID    `validate:"required"`
	ClientID           types.UserID    `validate:"required"`
	CanTakeMoreProblem bool
}

func (e NewChatEvent) Validate() error { return validator.Validator.Struct(e) }

// ChatClosedEvent is a signal about the problem is resolved.
type ChatClosedEvent struct {
	event              `gonstructor:"-"`
	EventID            types.EventID   `validate:"required"`
	RequestID          types.RequestID `validate:"required"`
	ChatID             types.ChatID    `validate:"required"`
	CanTakeMoreProblem bool
}

func (e ChatClosedEvent) Validate() error {
	return validator.Validator.Struct(e)
}
