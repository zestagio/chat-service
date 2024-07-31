package resolveproblem

import (
	"context"
	"errors"
	"fmt"
	"time"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	closechatjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/close-chat"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=resolveproblemmocks

var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrProblemNotFound = errors.New("problem not found")
)

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type messageRepository interface {
	CreateClientService(
		ctx context.Context,
		problemID types.ProblemID,
		chatID types.ChatID,
		msgBody string,
	) (*messagesrepo.Message, error)
}

type problemsRepository interface {
	GetAssignedProblemID(ctx context.Context, managerID types.UserID, chatID types.ChatID) (types.ProblemID, error)
	Resolve(ctx context.Context, problemID types.ProblemID) error
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	msgRepo      messageRepository  `option:"mandatory" validate:"required"`
	problemsRepo problemsRepository `option:"mandatory" validate:"required"`
	outbox       outboxService      `option:"mandatory" validate:"required"`
	txtor        transactor         `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	return UseCase{Options: opts}, opts.Validate()
}

func (u UseCase) Handle(ctx context.Context, req Request) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validate request: %w: %v", ErrInvalidRequest, err)
	}

	return u.txtor.RunInTx(ctx, func(ctx context.Context) error {
		problemID, err := u.problemsRepo.GetAssignedProblemID(ctx, req.ManagerID, req.ChatID)
		if err != nil {
			if errors.Is(err, problemsrepo.ErrProblemNotFound) {
				return ErrProblemNotFound
			}
			return fmt.Errorf("get assigned problem id: %w", err)
		}

		if err := u.problemsRepo.Resolve(ctx, problemID); err != nil {
			return fmt.Errorf("resolve problem: %v", err)
		}

		m, err := u.msgRepo.CreateClientService(
			ctx,
			problemID,
			req.ChatID,
			"Your question has been marked as resolved.\nThank you for being with us!",
		)
		if err != nil {
			return fmt.Errorf("create client service msg: %v", err)
		}

		_, err = u.outbox.Put(ctx, closechatjob.Name, simpleid.MustMarshal(m.ID), time.Now())
		if err != nil {
			return fmt.Errorf("create `close chat` job: %v", err)
		}

		return nil
	})
}
