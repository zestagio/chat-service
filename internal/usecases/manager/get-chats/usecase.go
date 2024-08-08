package getchats

import (
	"context"
	"fmt"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=getchatsmocks

type chatsRepository interface {
	GetChatsWithOpenProblems(ctx context.Context, managerID types.UserID) ([]chatsrepo.Chat, error)
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
		return Response{}, err
	}

	openChats, err := u.chatsRepo.GetChatsWithOpenProblems(ctx, req.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("get chats with open problems: %v", err)
	}

	result := make([]Chat, 0, len(openChats))
	for _, c := range openChats {
		result = append(result, Chat{
			ID:       c.ID,
			ClientID: c.ClientID,
		})
	}

	return Response{
		Chats: result,
	}, nil
}
