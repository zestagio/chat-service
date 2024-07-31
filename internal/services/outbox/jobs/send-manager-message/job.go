package sendmanagermessagejob

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	msgproducer "github.com/zestagio/chat-service/internal/services/msg-producer"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/job_mock.gen.go -package=sendmanagermessagejobmocks

const Name = "send-manager-message"

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

type messageProducer interface {
	ProduceMessage(ctx context.Context, message msgproducer.Message) error
}

type messageRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

type chatRepository interface {
	GetChatByID(ctx context.Context, chatID types.ChatID) (*chatsrepo.Chat, error)
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	eventStream eventStream       `option:"mandatory" validate:"required"`
	msgProducer messageProducer   `option:"mandatory" validate:"required"`
	chatsRepo   chatRepository    `option:"mandatory" validate:"required"`
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

	chat, err := j.chatsRepo.GetChatByID(ctx, msg.ChatID)
	if err != nil {
		return fmt.Errorf("get chat: %v", err)
	}

	if err := j.msgProducer.ProduceMessage(ctx, msgproducer.Message{
		ID:         msg.ID,
		ChatID:     msg.ChatID,
		Body:       msg.Body,
		FromClient: false,
	}); err != nil {
		return fmt.Errorf("produce message to queue: %v", err)
	}

	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() error {
		if err := j.eventStream.Publish(ctx, chat.ClientID, eventstream.NewNewMessageEvent(
			types.NewEventID(),
			msg.InitialRequestID,
			msg.ChatID,
			msg.ID,
			msg.AuthorID,
			msg.CreatedAt,
			msg.Body,
			msg.IsService,
		)); err != nil {
			return fmt.Errorf("publish NewMesaggeEvent to client: %v", err)
		}
		return nil
	})

	wg.Go(func() error {
		if err := j.eventStream.Publish(ctx, msg.AuthorID,
			eventstream.NewNewMessageEvent(
				types.NewEventID(),
				msg.InitialRequestID,
				msg.ChatID,
				msg.ID,
				msg.AuthorID,
				msg.CreatedAt,
				msg.Body,
				msg.IsService,
			)); err != nil {
			return fmt.Errorf("publish NewMessageEvent to manager: %v", err)
		}
		return nil
	})

	return wg.Wait()
}
