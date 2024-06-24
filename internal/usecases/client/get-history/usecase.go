package gethistory

import (
	"context"
	"errors"
	"fmt"

	"github.com/zestagio/chat-service/internal/cursor"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=gethistorymocks

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidCursor  = errors.New("invalid cursor")
)

type messagesRepository interface {
	GetClientChatMessages(
		ctx context.Context,
		clientID types.UserID,
		pageSize int,
		cursor *messagesrepo.Cursor,
	) ([]messagesrepo.Message, *messagesrepo.Cursor, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	msgRepo messagesRepository `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	return UseCase{Options: opts}, opts.Validate()
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := req.Validate(); err != nil {
		return Response{}, fmt.Errorf("validate request: %w: %v", ErrInvalidRequest, err)
	}

	var c *messagesrepo.Cursor
	if req.Cursor != "" {
		if err := cursor.Decode(req.Cursor, &c); err != nil {
			return Response{}, fmt.Errorf("decode cursor: %w: %v", ErrInvalidCursor, err)
		}
	}

	msgs, next, err := u.msgRepo.GetClientChatMessages(ctx, req.ClientID, req.PageSize, c)
	if err != nil {
		if errors.Is(err, messagesrepo.ErrInvalidCursor) {
			return Response{}, fmt.Errorf("get client chat messages: %w: %v", ErrInvalidCursor, err)
		}
		return Response{}, fmt.Errorf("get client chat messages: %v", err)
	}

	var nextCursor string
	if next != nil {
		data, err := cursor.Encode(next)
		if err != nil {
			return Response{}, fmt.Errorf("encode next cursor: %v", err)
		}
		nextCursor = data
	}

	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, Message{
			ID:         m.ID,
			AuthorID:   m.AuthorID,
			Body:       m.Body,
			CreatedAt:  m.CreatedAt,
			IsReceived: m.IsVisibleForManager && !m.IsBlocked,
			IsBlocked:  m.IsBlocked,
			IsService:  m.IsService,
		})
	}

	return Response{
		Messages:   result,
		NextCursor: nextCursor,
	}, nil
}
