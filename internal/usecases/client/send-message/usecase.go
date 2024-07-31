package sendmessage

import (
	"context"
	"errors"
	"fmt"
	"time"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	sendclientmessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-client-message"
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

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type problemsRepository interface {
	CreateIfNotExists(ctx context.Context, chatID types.ChatID) (types.ProblemID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	chatsRepo    chatsRepository    `option:"mandatory" validate:"required"`
	msgRepo      messagesRepository `option:"mandatory" validate:"required"`
	outBox       outboxService      `option:"mandatory" validate:"required"`
	problemsRepo problemsRepository `option:"mandatory" validate:"required"`
	txtor        transactor         `option:"mandatory" validate:"required"`
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

	var msg *messagesrepo.Message

	if err := u.txtor.RunInTx(ctx, func(ctx context.Context) error {
		m, err := u.msgRepo.GetMessageByRequestID(ctx, req.ID)
		if nil == err {
			msg = m
			return nil
		}
		if !errors.Is(err, messagesrepo.ErrMsgNotFound) {
			return fmt.Errorf("get msg by initial request id: %v", err)
		}

		chatID, err := u.chatsRepo.CreateIfNotExists(ctx, req.ClientID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrChatNotCreated, err)
		}

		problemID, err := u.problemsRepo.CreateIfNotExists(ctx, chatID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrProblemNotCreated, err)
		}

		m, err = u.msgRepo.CreateClientVisible(ctx, req.ID, problemID, chatID, req.ClientID, req.MessageBody)
		if err != nil {
			return fmt.Errorf("create client visible message: %v", err)
		}

		_, err = u.outBox.Put(ctx, sendclientmessagejob.Name, simpleid.MustMarshal(m.ID), time.Now())
		if err != nil {
			return fmt.Errorf("create `send client message` job: %v", err)
		}

		msg = m
		return nil
	}); err != nil {
		return Response{}, fmt.Errorf("`send client message` tx: %w", err)
	}

	return Response{
		AuthorID:  msg.AuthorID,
		MessageID: msg.ID,
		CreatedAt: msg.CreatedAt,
	}, nil
}
