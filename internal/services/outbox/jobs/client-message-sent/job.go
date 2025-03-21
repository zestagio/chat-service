package clientmessagesentjob

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

const Name = "client-message-sent"

type chatsRepository interface {
	GetChatManager(ctx context.Context, chatID types.ChatID) (types.UserID, error)
}

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

type messageRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	chatsRepo   chatsRepository   `option:"mandatory" validate:"required"`
	eventStream eventStream       `option:"mandatory" validate:"required"`
	msgRepo     messageRepository `option:"mandatory" validate:"required"`
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

	msgID, err := simpleid.Unmarshal[types.MessageID](payload)
	if err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	msg, err := j.msgRepo.GetMessageByID(ctx, msgID)
	if err != nil {
		return fmt.Errorf("get message: %v", err)
	}

	wg, ctx := errgroup.WithContext(ctx)

	// Send update to client.
	wg.Go(func() error {
		if err := j.eventStream.Publish(ctx, msg.AuthorID,
			eventstream.NewMessageSentEvent(
				types.NewEventID(),
				msg.InitialRequestID,
				msg.ID,
			),
		); err != nil {
			return fmt.Errorf("publish MessageSentEvent to client: %v", err)
		}
		return nil
	})

	// Send update to manager.
	wg.Go(func() error {
		managerID, err := j.chatsRepo.GetChatManager(ctx, msg.ChatID)
		if errors.Is(err, chatsrepo.ErrChatWithoutManager) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("get chat manager: %v", err)
		}

		if err := j.eventStream.Publish(ctx, managerID, eventstream.NewNewMessageEvent(
			types.NewEventID(),
			msg.InitialRequestID,
			msg.ChatID,
			msg.ID,
			msg.AuthorID,
			msg.CreatedAt,
			msg.Body,
			false,
		)); err != nil {
			return fmt.Errorf("publish NewMessageEvent to manager: %v", err)
		}
		return nil
	})

	return wg.Wait()
}
