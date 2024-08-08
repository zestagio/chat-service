package managerscheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	managerpool "github.com/zestagio/chat-service/internal/services/manager-pool"
	managerassignedjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/manager-assigned-to-problem"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

const serviceName = "manager-scheduler"

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
	GetProblemsWithoutManager(ctx context.Context, lim int) ([]problemsrepo.Problem, error)
	SetManagerForProblem(ctx context.Context, problemID types.ProblemID, managerID types.UserID) error
	GetProblemInitialRequestID(ctx context.Context, pID types.ProblemID) (types.RequestID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	period time.Duration `option:"mandatory" validate:"min=100ms,max=1m"`

	mngrPool     managerpool.Pool   `option:"mandatory" validate:"required"`
	msgRepo      messagesRepository `option:"mandatory" validate:"required"`
	outBox       outboxService      `option:"mandatory" validate:"required"`
	problemsRepo problemsRepository `option:"mandatory" validate:"required"`
	txtor        transactor         `option:"mandatory" validate:"required"`
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
		if err := s.assignProblemManagers(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Error("assign problem managers", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.period):
		}
	}
}

func (s *Service) assignProblemManagers(ctx context.Context) error {
	lim := s.mngrPool.Size()
	if lim == 0 {
		s.logger.Debug("no available managers")
		return nil
	}

	problems, err := s.problemsRepo.GetProblemsWithoutManager(ctx, lim)
	if err != nil {
		return fmt.Errorf("get problems without manager: %v", err)
	}
	if len(problems) == 0 {
		s.logger.Debug("no new problems")
		return nil
	}

	for _, p := range problems {
		if err := s.assignManagerToProblem(ctx, p); err != nil {
			return fmt.Errorf("assign manager to problem %s: %v", p.ID, err)
		}
	}
	return nil
}

func (s *Service) assignManagerToProblem(ctx context.Context, p problemsrepo.Problem) (errReturned error) {
	managerID, err := s.mngrPool.Get(ctx)
	if err != nil {
		return fmt.Errorf("get manager from pool: %v", err)
	}
	defer func() {
		if errReturned != nil {
			// Specially left (for teaching purposes) architectural kostyl.
			if err := s.mngrPool.Put(ctx, managerID); err != nil {
				s.logger.Error("cannot put manager back in the pool",
					zap.Error(err),
					zap.Stringer("manager_id", managerID))
			}
		}
	}()

	return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.problemsRepo.SetManagerForProblem(ctx, p.ID, managerID); err != nil {
			return fmt.Errorf("set problem manager: %v", err)
		}

		reqID, err := s.problemsRepo.GetProblemInitialRequestID(ctx, p.ID)
		if err != nil {
			return fmt.Errorf("get initial request: %v", err)
		}

		notifyText := fmt.Sprintf("Manager %s will answer you", managerID)
		notifyMsgID, err := s.msgRepo.CreateServiceMessageForClient(ctx, reqID, p.ID, p.ChatID, notifyText)
		if err != nil {
			return fmt.Errorf("create service message for client: %v", err)
		}

		if _, err := s.outBox.Put(ctx, managerassignedjob.Name, simpleid.MustMarshal(notifyMsgID), time.Now()); err != nil {
			return fmt.Errorf("put job: %v", err)
		}

		s.logger.Info("set manager for problem",
			zap.Stringer("manager_id", managerID),
			zap.Stringer("problem_id", p.ID),
		)
		return nil
	})
}
