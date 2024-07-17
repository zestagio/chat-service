package clientmessagesentjob

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/job_mock.gen.go -package=clientmessagesentjobmocks

const Name = "client-message-sent"

type messageRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	msgRepo     messageRepository `option:"mandatory" validate:"required"`
	eventStream eventStream       `option:"mandatory" validate:"required"`
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

	p, err := jobs.UnmarshalPayload(payload)
	if err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	m, err := j.msgRepo.GetMessageByID(ctx, p.MessageID)
	if err != nil {
		return fmt.Errorf("get message: %v", err)
	}

	if err := j.eventStream.Publish(ctx, m.AuthorID, eventstream.NewMessageSentEvent(
		types.NewEventID(),
		m.InitialRequestID,
		m.ID,
	)); err != nil {
		return fmt.Errorf("publish message to event stream: %v", err)
	}
	return nil
}
