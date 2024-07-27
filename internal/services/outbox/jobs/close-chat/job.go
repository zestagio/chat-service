package closechatjob

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/job_mock.gen.go -package=closechatjobmocks

const (
	Name         = "close-chat"
	CloseMsgBody = "Your question has been marked as resolved.\nThank you for being with us!"
)

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

type chatsRepository interface {
	GetChatByID(ctx context.Context, chatID types.ChatID) (*chatsrepo.Chat, error)
}

type problemRepository interface {
	GetProblemByID(ctx context.Context, problemID types.ProblemID) (*problemsrepo.Problem, error)
	GetProblemRequestID(ctx context.Context, problemID types.ProblemID) (types.RequestID, error)
}

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	eventStream eventStream        `option:"mandatory" validate:"required"`
	chatsRepo   chatsRepository    `option:"mandatory" validate:"required"`
	problemRepo problemRepository  `option:"mandatory" validate:"required"`
	managerLoad managerLoadService `option:"mandatory" validate:"required"`
}

type Job struct {
	outbox.DefaultJob
	Options
	logger *zap.Logger
}

func Must(opts Options) *Job {
	j, err := New(opts)
	if err != nil {
		panic(err)
	}
	return j
}

func New(opts Options) (*Job, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}
	return &Job{
		Options: opts,
		logger:  zap.L().Named("job." + Name),
	}, nil
}

func (j *Job) Name() string {
	return Name
}

func (j *Job) Handle(ctx context.Context, payload string) error {
	j.logger.Info("start processing", zap.String("payload", payload))

	pID, err := simpleid.Unmarshal[types.ProblemID](payload)
	if err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	problem, err := j.problemRepo.GetProblemByID(ctx, pID)
	if err != nil {
		return fmt.Errorf("get problem: %v", err)
	}

	initReqID, err := j.problemRepo.GetProblemRequestID(ctx, problem.ID)
	if err != nil {
		return fmt.Errorf("get problem request id: %v", err)
	}

	chat, err := j.chatsRepo.GetChatByID(ctx, problem.ChatID)
	if err != nil {
		return fmt.Errorf("get chat: %v", err)
	}

	canManagerTakeProblem, err := j.managerLoad.CanManagerTakeProblem(ctx, problem.ManagerID)
	if err != nil {
		return fmt.Errorf("can manager take problem: %v", err)
	}

	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() error {
		if err := j.eventStream.Publish(ctx, chat.ClientID, eventstream.NewNewMessageEvent(
			types.NewEventID(),
			initReqID,
			problem.ChatID,
			types.NewMessageID(),
			types.UserIDNil,
			time.Now(),
			CloseMsgBody,
			true,
		)); err != nil {
			return fmt.Errorf("publish NewMesaggeEvent to client: %v", err)
		}
		return nil
	})

	wg.Go(func() error {
		if err := j.eventStream.Publish(ctx, problem.ManagerID, eventstream.NewChatClosedEvent(
			types.NewEventID(),
			initReqID,
			problem.ChatID,
			canManagerTakeProblem,
		)); err != nil {
			return fmt.Errorf("publish ChatClosedEvent to manager: %v", err)
		}
		return nil
	})

	return wg.Wait()
}
