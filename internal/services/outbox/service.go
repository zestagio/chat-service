package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	jobsrepo "github.com/zestagio/chat-service/internal/repositories/jobs"
	"github.com/zestagio/chat-service/internal/types"
)

const serviceName = "outbox"

type jobsRepository interface {
	CreateJob(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
	FindAndReserveJob(ctx context.Context, until time.Time) (jobsrepo.Job, error)
	CreateFailedJob(ctx context.Context, name, payload, reason string) error
	DeleteJob(ctx context.Context, jobID types.JobID) error
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	workers    int           `option:"mandatory" validate:"min=1,max=32"`
	idleTime   time.Duration `option:"mandatory" validate:"min=100ms,max=10s"`
	reserveFor time.Duration `option:"mandatory" validate:"min=1s,max=10m"`

	jobsRepo jobsRepository `option:"mandatory" validate:"required"`
	txtor    transactor     `option:"mandatory" validate:"required"`
}

type Service struct {
	Options
	jobs map[string]Job
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}
	return &Service{
		Options: opts,
		jobs:    map[string]Job{},
	}, nil
}

func (s *Service) RegisterJob(job Job) error {
	if _, ok := s.jobs[job.Name()]; ok {
		return fmt.Errorf("job %q already registered", job.Name())
	}

	s.jobs[job.Name()] = job
	return nil
}

func (s *Service) MustRegisterJob(job Job) {
	if err := s.RegisterJob(job); err != nil {
		panic(fmt.Errorf("register job: %v", err))
	}
}

func (s *Service) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	for i := 0; i < s.workers; i++ {
		logger := zap.L().Named(serviceName).With(zap.Int("worker", i+1))
		eg.Go(func() error {
			for {
				// Process all available jobs in one go.
				if err := s.processAvailableJobs(ctx, logger); err != nil {
					if ctx.Err() != nil {
						return nil //nolint:nilerr // graceful exit
					}
					logger.Warn("process jobs error", zap.Error(err))
					return err
				}

				select {
				case <-ctx.Done():
					return nil
				case <-time.After(s.idleTime):
				}
			}
		})
	}

	return eg.Wait()
}

func (s *Service) processAvailableJobs(ctx context.Context, log *zap.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := s.findAndProcessJob(ctx, log); err != nil {
			if errors.Is(err, jobsrepo.ErrNoJobs) {
				log.Debug("no jobs found to process")
				return nil
			}
			return err
		}
	}
}

func (s *Service) findAndProcessJob(ctx context.Context, log *zap.Logger) error {
	job, err := s.jobsRepo.FindAndReserveJob(ctx, time.Now().Local().Add(s.reserveFor))
	if err != nil {
		return fmt.Errorf("find and reserve job: %w", err)
	}

	log = log.With(
		zap.String("job_name", job.Name),
		zap.Stringer("job_id", job.ID),
		zap.Int("attempt_number", job.Attempts))

	j, ok := s.jobs[job.Name]
	if !ok {
		log.Warn("drop to dlq: job is not registered")
		return s.dlq(ctx, job.ID, job.Name, job.Payload, "unknown job")
	}

	func() {
		ctx, cancel := context.WithTimeout(ctx, j.ExecutionTimeout())
		defer cancel()

		err = j.Handle(ctx, job.Payload)
	}()

	if err != nil {
		log.Warn("handle job error", zap.Error(err))

		if job.Attempts >= j.MaxAttempts() {
			log.Warn("drop to dlq: job max attempts exceeded")
			return s.dlq(
				ctx,
				job.ID,
				job.Name,
				job.Payload,
				fmt.Sprintf("max attempts exceeded: %v", err),
			)
		}
		return nil
	}

	//nolint:contextcheck // intentionally delete job with context.Background() to avoid case when job is handled,
	// but ctx is already closed before deleting.
	if err := s.jobsRepo.DeleteJob(context.Background(), job.ID); err != nil {
		log.Warn("delete job error", zap.Error(err))
	}
	return nil
}

func (s *Service) dlq(ctx context.Context, jobID types.JobID, name, payload, reason string) error {
	return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.jobsRepo.CreateFailedJob(ctx, name, payload, reason); err != nil {
			return fmt.Errorf("create failed job: %v", err)
		}

		if err := s.jobsRepo.DeleteJob(ctx, jobID); err != nil {
			return fmt.Errorf("delete job: %v", err)
		}

		return nil
	})
}
