package sendmessage

import (
	"context"
	"errors"
	"fmt"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=sendmessagemocks

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrChatNotCreated    = errors.New("chat not created")
	ErrProblemNotCreated = errors.New("problem not created")
)

type chatsRepository interface {
	CreateIfNotExists(ctx context.Context, userID types.UserID) (types.ChatID, error)
}

type messagesRepository interface {
	GetMessageByRequestID(ctx context.Context, reqID types.RequestID) (*messagesrepo.Message, error)
	CreateClientVisible(
		ctx context.Context,
		reqID types.RequestID,
		problemID types.ProblemID,
		chatID types.ChatID,
		authorID types.UserID,
		msgBody string,
	) (*messagesrepo.Message, error)
}

type problemsRepository interface {
	CreateIfNotExists(ctx context.Context, chatID types.ChatID) (types.ProblemID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	chatRepo    chatsRepository    `option:"mandatory" validation:"required"`
	msgRepo     messagesRepository `option:"mandatory" validation:"required"`
	problemRepo problemsRepository `option:"mandatory" validation:"required"`
	txtor       transactor         `option:"mandatory" validation:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	return UseCase{Options: opts}, opts.Validate()
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := req.Validate(); err != nil {
		return Response{}, ErrInvalidRequest
	}

	var message *messagesrepo.Message

	err := u.txtor.RunInTx(ctx, func(ctx context.Context) error {
		var err error

		message, err = u.msgRepo.GetMessageByRequestID(ctx, req.ID)
		if err == nil {
			return nil
		}
		if !errors.Is(err, messagesrepo.ErrMsgNotFound) {
			return fmt.Errorf("get msg by request id: %v", err)
		}

		chatID, err := u.chatRepo.CreateIfNotExists(ctx, req.ClientID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrChatNotCreated, err)
		}

		problemID, err := u.problemRepo.CreateIfNotExists(ctx, chatID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrProblemNotCreated, err)
		}

		message, err = u.msgRepo.CreateClientVisible(ctx, req.ID, problemID, chatID, req.ClientID, req.MessageBody)

		return err
	})
	if err != nil {
		return Response{}, err
	}

	return Response{
		MessageID: message.ID,
		AuthorID:  message.AuthorID,
		CreatedAt: message.CreatedAt,
	}, nil
}
