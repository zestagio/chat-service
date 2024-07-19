package managerassignedtoproblemjob

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

const Name = "manager-assigned-to-problem"

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

type chatRepository interface {
	GetChatByID(ctx context.Context, chatID types.ChatID) (*chatsrepo.Chat, error)
}

type problemRepository interface {
	GetProblemByID(ctx context.Context, problemID types.ProblemID) (*problemsrepo.Problem, error)
	GetProblemRequestID(ctx context.Context, problemID types.ProblemID) (types.RequestID, error)
}

type messageRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	eventStream eventStream        `option:"mandatory" validate:"required"`
	chatRepo    chatRepository     `option:"mandatory" validate:"required"`
	problemRepo problemRepository  `option:"mandatory" validate:"required"`
	msgRepo     messageRepository  `option:"mandatory" validate:"required"`
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
		logger:  zap.L().Named(Name),
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

	chat, err := j.chatRepo.GetChatByID(ctx, msg.ChatID)
	if err != nil {
		return fmt.Errorf("get chat: %v", err)
	}

	problem, err := j.problemRepo.GetProblemByID(ctx, msg.ProblemID)
	if err != nil {
		return fmt.Errorf("get problem: %v", err)
	}

	initReqID, err := j.problemRepo.GetProblemRequestID(ctx, problem.ID)
	if err != nil {
		return fmt.Errorf("get problem request id: %v", err)
	}

	if err := j.eventStream.Publish(ctx, chat.ClientID, eventstream.NewNewMessageEvent(
		types.NewEventID(),
		initReqID,
		msg.ChatID,
		msg.ID,
		types.UserIDNil,
		msg.CreatedAt,
		msg.Body,
		msg.IsService,
	)); err != nil {
		return fmt.Errorf("produce message to queue: %v", err)
	}

	canTakeMoreProblems, err := j.managerLoad.CanManagerTakeProblem(ctx, problem.ManagerID)
	if err != nil {
		return fmt.Errorf("can manager take problem: %v", err)
	}

	if err := j.eventStream.Publish(ctx, problem.ManagerID, eventstream.NewNewChatEvent(
		types.NewEventID(),
		initReqID,
		chat.ID,
		chat.ClientID,
		canTakeMoreProblems,
	)); err != nil {
		return fmt.Errorf("produce message to queue: %v", err)
	}

	return nil
}
