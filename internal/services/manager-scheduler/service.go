package managerscheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	managerpool "github.com/zestagio/chat-service/internal/services/manager-pool"
	managerassignedtoproblemjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/manager-assigned-to-problem"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

const serviceName = "manager-scheduler"

//go:generate mockgen -source=$GOFILE -destination=mocks/service_mock.gen.go -package=managerschedulermocks

type problemRepository interface {
	GetAvailableProblems(ctx context.Context) ([]problemsrepo.Problem, error)
	SetManagerForProblem(ctx context.Context, problemID types.ProblemID, managerID types.UserID) error
	GetProblemRequestID(ctx context.Context, problemID types.ProblemID) (types.RequestID, error)
}

type messageRepository interface {
	CreateClientService(
		ctx context.Context,
		problemID types.ProblemID,
		chatID types.ChatID,
		msgBody string,
	) (*messagesrepo.Message, error)
}

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	period time.Duration `option:"mandatory" validate:"min=100ms,max=1m"`

	mngrPool     managerpool.Pool  `option:"mandatory" validate:"required"`
	msgRepo      messageRepository `option:"mandatory" validate:"required"`
	problemsRepo problemRepository `option:"mandatory" validate:"required"`
	outBox       outboxService     `option:"mandatory" validate:"required"`
	txtor        transactor        `option:"mandatory" validate:"required"`
}

type Service struct {
	Options

	logger *zap.Logger
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	return &Service{
		Options: opts,
		logger:  zap.L().Named(serviceName),
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	for {
		err := s.scheduleManagersToProblems(ctx)
		if err != nil && !errors.Is(err, managerpool.ErrNoAvailableManagers) {
			return fmt.Errorf("schedule managers to problem: %v", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.period):
		}
	}
}

func (s *Service) scheduleManagersToProblems(ctx context.Context) error {
	problems, err := s.problemsRepo.GetAvailableProblems(ctx)
	if err != nil {
		return fmt.Errorf("get available problems: %v", err)
	}

	for _, p := range problems {
		managerID, err := s.mngrPool.Get(ctx)
		if err != nil {
			return fmt.Errorf("get manager from pool: %w", err)
		}

		if err := s.setManager(ctx, managerID, p); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) setManager(ctx context.Context, managerID types.UserID, p problemsrepo.Problem) error {
	return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.problemsRepo.SetManagerForProblem(ctx, p.ID, managerID); err != nil {
			return fmt.Errorf("set manager for problem: %v", err)
		}

		m, err := s.msgRepo.CreateClientService(
			ctx,
			p.ID,
			p.ChatID,
			fmt.Sprintf("Manager %s will answer you", managerID.String()),
		)
		if err != nil {
			return fmt.Errorf("create client service msg: %v", err)
		}

		_, err = s.outBox.Put(ctx, managerassignedtoproblemjob.Name, simpleid.MustMarshal(m.ID), time.Now())
		if err != nil {
			return fmt.Errorf("create `manager assigned to problem` job: %v", err)
		}

		return nil
	})
}
