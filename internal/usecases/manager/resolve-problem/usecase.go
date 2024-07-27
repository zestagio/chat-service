package resolveproblem

import (
	"context"
	"errors"
	"fmt"
	"time"

	closechatjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/close-chat"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=resolveproblemmocks

var ErrInvalidRequest = errors.New("invalid request")

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type problemsRepository interface {
	Resolve(ctx context.Context, managerID types.UserID, chatID types.ChatID) (types.ProblemID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
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
		problemID, err := u.problemsRepo.Resolve(ctx, req.ManagerID, req.ChatID)
		if err != nil {
			return fmt.Errorf("resolve problem: %v", err)
		}

		_, err = u.outbox.Put(ctx, closechatjob.Name, simpleid.MustMarshal(problemID), time.Now())
		if err != nil {
			return fmt.Errorf("create `close chat` job: %v", err)
		}

		return nil
	})
}
