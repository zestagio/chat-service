package sendmessage

import (
	"context"
	"fmt"
	"time"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	sendmanagermessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-manager-message"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=sendmessagemocks

type messagesRepository interface {
	CreateFullVisible(
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
	GetAssignedProblemID(ctx context.Context, managerID types.UserID, chatID types.ChatID) (types.ProblemID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
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
		return Response{}, err
	}

	problemID, err := u.problemsRepo.GetAssignedProblemID(ctx, req.ManagerID, req.ChatID)
	if err != nil {
		return Response{}, fmt.Errorf("get assigned problem: %v", err)
	}

	var msg *messagesrepo.Message

	if err := u.txtor.RunInTx(ctx, func(ctx context.Context) error {
		m, err := u.msgRepo.CreateFullVisible(ctx, req.ID, problemID, req.ChatID, req.ManagerID, req.MessageBody)
		if err != nil {
			return fmt.Errorf("create full visible message: %v", err)
		}

		_, err = u.outBox.Put(ctx, sendmanagermessagejob.Name, simpleid.MustMarshal(m.ID), time.Now())
		if err != nil {
			return fmt.Errorf("create `send manager message` job: %v", err)
		}

		msg = m
		return nil
	}); err != nil {
		return Response{}, fmt.Errorf("`send manager message` tx: %v", err)
	}

	return Response{
		MessageID: msg.ID,
		CreatedAt: msg.CreatedAt,
	}, nil
}
