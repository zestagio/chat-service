package getchats

import (
	"context"
	"errors"
	"fmt"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=getchatsmocks

var ErrInvalidRequest = errors.New("invalid request")

type chatsRepository interface {
	GetManagerChatsWithProblems(ctx context.Context, managerID types.UserID) ([]chatsrepo.Chat, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	chatsRepo chatsRepository `option:"mandatory" validate:"required"`
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

	chats, err := u.chatsRepo.GetManagerChatsWithProblems(ctx, req.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("get manager chats: %w", err)
	}

	result := make([]Chat, 0, len(chats))
	for _, chat := range chats {
		result = append(result, Chat{
			ID:       chat.ID,
			ClientID: chat.ClientID,
		})
	}

	return Response{
		Chats: result,
	}, nil
}
