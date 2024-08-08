package sendmanagermessagejob

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	msgproducer "github.com/zestagio/chat-service/internal/services/msg-producer"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

const Name = "send-manager-message"

type chatsRepository interface {
	GetChatClient(ctx context.Context, chatID types.ChatID) (types.UserID, error)
}

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

type messageProducer interface {
	ProduceMessage(ctx context.Context, message msgproducer.Message) error
}

type messageRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	chatsRepo   chatsRepository   `option:"mandatory" validate:"required"`
	eventStream eventStream       `option:"mandatory" validate:"required"`
	msgProducer messageProducer   `option:"mandatory" validate:"required"`
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

	m, err := j.msgRepo.GetMessageByID(ctx, msgID)
	if err != nil {
		return fmt.Errorf("get message: %v", err)
	}

	clientID, err := j.chatsRepo.GetChatClient(ctx, m.ChatID)
	if err != nil {
		return fmt.Errorf("get chat client: %v", err)
	}

	if err := j.msgProducer.ProduceMessage(ctx, msgproducer.Message{
		ID:         m.ID,
		ChatID:     m.ChatID,
		Body:       m.Body,
		FromClient: false,
	}); err != nil {
		return fmt.Errorf("produce message to queue: %v", err)
	}

	wg, ctx := errgroup.WithContext(ctx)

	// Send update to client.
	wg.Go(func() error {
		return j.eventStream.Publish(ctx, clientID, eventstream.NewNewMessageEvent(
			types.NewEventID(),
			m.InitialRequestID,
			m.ChatID,
			m.ID,
			m.AuthorID,
			m.CreatedAt,
			m.Body,
			m.IsService,
		))
	})

	// Send update to manager (to oneself).
	wg.Go(func() error {
		return j.eventStream.Publish(ctx, m.AuthorID, eventstream.NewNewMessageEvent(
			types.NewEventID(),
			m.InitialRequestID,
			m.ChatID,
			m.ID,
			m.AuthorID,
			m.CreatedAt,
			m.Body,
			m.IsService,
		))
	})

	return wg.Wait()
}
