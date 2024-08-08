package getchathistory

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/cursor"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=getchathistorymocks

type messagesRepository interface {
	GetProblemMessages(
		ctx context.Context,
		problemID types.ProblemID,
		pageSize int,
		cursor *messagesrepo.Cursor,
	) ([]messagesrepo.Message, *messagesrepo.Cursor, error)
}

type problemsRepository interface {
	GetAssignedProblemID(ctx context.Context, managerID types.UserID, chatID types.ChatID) (types.ProblemID, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	msgRepo      messagesRepository `option:"mandatory" validate:"required"`
	problemsRepo problemsRepository `option:"mandatory" validate:"required"`
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

	var c *messagesrepo.Cursor
	if req.Cursor != "" {
		if err := cursor.Decode(req.Cursor, &c); err != nil {
			return Response{}, fmt.Errorf("decode cursor: %v", err)
		}
	}

	currentProblemID, err := u.problemsRepo.GetAssignedProblemID(ctx, req.ManagerID, req.ChatID)
	if err != nil {
		return Response{}, fmt.Errorf("get assigned problem: %v", err)
	}

	msgs, next, err := u.msgRepo.GetProblemMessages(ctx, currentProblemID, req.PageSize, c)
	if err != nil {
		return Response{}, fmt.Errorf("get manager chat messages: %v", err)
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
			ID:        m.ID,
			AuthorID:  m.AuthorID,
			Body:      m.Body,
			CreatedAt: m.CreatedAt,
		})
	}

	return Response{
		Messages:   result,
		NextCursor: nextCursor,
	}, nil
}
