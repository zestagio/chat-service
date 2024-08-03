package managerassignedtoproblemjob

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

const Name = "manager-assigned-to-problem"

type chatsRepository interface {
	GetChatClient(ctx context.Context, chatID types.ChatID) (types.UserID, error)
	GetChatManager(ctx context.Context, chatID types.ChatID) (types.UserID, error)
}

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

type messageRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	chatsRepo   chatsRepository    `option:"mandatory" validate:"required"`
	eventStream eventStream        `option:"mandatory" validate:"required"`
	msgRepo     messageRepository  `option:"mandatory" validate:"required"`
	mLoadSvc    managerLoadService `option:"mandatory" validate:"required"`
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

	serviceMsgID, err := simpleid.Unmarshal[types.MessageID](payload)
	if err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	serviceMsg, err := j.msgRepo.GetMessageByID(ctx, serviceMsgID)
	if err != nil {
		return fmt.Errorf("get message: %v", err)
	}

	clientID, err := j.chatsRepo.GetChatClient(ctx, serviceMsg.ChatID)
	if err != nil {
		return fmt.Errorf("get client: %v", err)
	}

	managerID, err := j.chatsRepo.GetChatManager(ctx, serviceMsg.ChatID)
	if err != nil {
		return fmt.Errorf("get manager: %v", err)
	}

	canTakeMore, err := j.mLoadSvc.CanManagerTakeProblem(ctx, managerID)
	if err != nil {
		return fmt.Errorf("manager load service call: %v", err)
	}

	wg, ctx := errgroup.WithContext(ctx)

	// Send update to client.
	wg.Go(func() error {
		if err := j.eventStream.Publish(ctx, clientID,
			eventstream.NewNewMessageEvent(
				types.NewEventID(),
				serviceMsg.InitialRequestID,
				serviceMsg.ChatID,
				serviceMsg.ID,
				types.UserIDNil,
				serviceMsg.CreatedAt,
				serviceMsg.Body,
				true,
			),
		); err != nil {
			return fmt.Errorf("publish service NewMessageEvent: %v", err)
		}
		return nil
	})

	// Send update to manager.
	wg.Go(func() error {
		if err := j.eventStream.Publish(ctx, managerID,
			eventstream.NewNewChatEvent(
				types.NewEventID(),
				serviceMsg.InitialRequestID,
				serviceMsg.ChatID,
				clientID,
				canTakeMore,
			),
		); err != nil {
			return fmt.Errorf("publish NewChatEvent: %v", err)
		}
		return nil
	})

	return wg.Wait()
}
