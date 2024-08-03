package resolveproblem

import (
	"context"
	"errors"
	"fmt"
	"time"

	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	problemresolvedjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/problem-resolved"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=resolveproblemmocks

var ErrAssignedProblemNotFound = errors.New("assigned problem not found")

const notifyText = `Your question has been marked as resolved.
Thank you for being with us!`

type messagesRepository interface {
	CreateServiceMessageForClient(
		ctx context.Context,
		reqID types.RequestID,
		problemID types.ProblemID,
		chatID types.ChatID,
		msgBody string,
	) (types.MessageID, error)
}

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type problemsRepository interface {
	GetAssignedProblemID(ctx context.Context, managerID types.UserID, chatID types.ChatID) (types.ProblemID, error)
	ResolveProblem(ctx context.Context, requestID types.RequestID, problemID types.ProblemID) error
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
		if errors.Is(err, problemsrepo.ErrAssignedProblemNotFound) {
			return Response{}, ErrAssignedProblemNotFound
		}
		return Response{}, fmt.Errorf("get assigned problem: %v", err)
	}

	if err := u.txtor.RunInTx(ctx, func(ctx context.Context) error {
		if err := u.problemsRepo.ResolveProblem(ctx, req.ID, problemID); err != nil {
			return fmt.Errorf("resolve problem: %v", err)
		}

		if _, err := u.msgRepo.CreateServiceMessageForClient(ctx, req.ID, problemID, req.ChatID, notifyText); err != nil {
			return fmt.Errorf("create service message for client: %v", err)
		}

		_, err = u.outBox.Put(ctx, problemresolvedjob.Name, simpleid.MustMarshal(req.ID), time.Now())
		return err
	}); err != nil {
		return Response{}, fmt.Errorf("resolve problem tx: %v", err)
	}
	return Response{}, nil
}
