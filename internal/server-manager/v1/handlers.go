package managerv1

import (
	"context"
	"fmt"

	canreceiveproblems "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems"
	freehandssignal "github.com/zestagio/chat-service/internal/usecases/manager/free-hands-signal"
	getchathistory "github.com/zestagio/chat-service/internal/usecases/manager/get-chat-history"
	getchats "github.com/zestagio/chat-service/internal/usecases/manager/get-chats"
	resolveproblem "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/manager/send-message"
)

var _ ServerInterface = (*Handlers)(nil)

//go:generate mockgen -source=$GOFILE -destination=mocks/handlers_mocks.gen.go -package=managerv1mocks

type canReceiveProblemsUseCase interface {
	Handle(ctx context.Context, req canreceiveproblems.Request) (canreceiveproblems.Response, error)
}

type freeHandsSignalUseCase interface {
	Handle(ctx context.Context, req freehandssignal.Request) (freehandssignal.Response, error)
}

type getChatsUseCase interface {
	Handle(ctx context.Context, req getchats.Request) (getchats.Response, error)
}

type getChatHistoryUseCase interface {
	Handle(ctx context.Context, req getchathistory.Request) (getchathistory.Response, error)
}

type sendMessageUseCase interface {
	Handle(ctx context.Context, req sendmessage.Request) (sendmessage.Response, error)
}

type resolveProblemUseCase interface {
	Handle(ctx context.Context, req resolveproblem.Request) error
}

//go:generate options-gen -out-filename=handlers.gen.go -from-struct=Options
type Options struct {
	canReceiveProblems    canReceiveProblemsUseCase `option:"mandatory" validate:"required"`
	freeHandsSignal       freeHandsSignalUseCase    `option:"mandatory" validate:"required"`
	getChatsUseCase       getChatsUseCase           `option:"mandatory" validate:"required"`
	getChatHistoryUseCase getChatHistoryUseCase     `option:"mandatory" validate:"required"`
	sendMessageUseCase    sendMessageUseCase        `option:"mandatory" validate:"required"`
	resolveProblemUseCase resolveProblemUseCase     `option:"mandatory" validate:"required"`
}

type Handlers struct {
	Options
}

func NewHandlers(opts Options) (Handlers, error) {
	if err := opts.Validate(); err != nil {
		return Handlers{}, fmt.Errorf("validate options: %v", err)
	}
	return Handlers{Options: opts}, nil
}
