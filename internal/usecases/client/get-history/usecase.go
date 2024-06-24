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
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("validate options: %v", err)
	}
	return UseCase{opts}, nil
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	resp := Response{}

	if err := req.Validate(); err != nil {
		return resp, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	var cur *messagesrepo.Cursor
	if req.Cursor != "" {
		if err := cursor.Decode(req.Cursor, &cur); err != nil {
			return resp, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
		}
	}

	messages, nextCur, err := u.Options.msgRepo.GetClientChatMessages(ctx, req.ClientID, req.PageSize, cur)
	if err != nil {
		if errors.Is(err, messagesrepo.ErrInvalidCursor) {
			return resp, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
		}
		return resp, err
	}

	resp.Messages = make([]Message, 0, len(messages))
	for _, msg := range messages {
		resp.Messages = append(resp.Messages, Message{
			ID:                  msg.ID,
			AuthorID:            msg.AuthorID,
			Body:                msg.Body,
			IsVisibleForManager: msg.IsVisibleForManager,
			IsBlocked:           msg.IsBlocked,
			IsReceived:          msg.IsVisibleForManager && !msg.IsBlocked,
			IsService:           msg.IsService,
			CreatedAt:           msg.CreatedAt,
		})
	}

	if nextCur != nil {
		resp.NextCursor, err = cursor.Encode(nextCur)
		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}
