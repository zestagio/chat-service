package managerv1

import (
	"context"
	"fmt"

	canreceiveproblems "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems"
	freehands "github.com/zestagio/chat-service/internal/usecases/manager/free-hands"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/handlers_mocks.gen.go -package=managerv1mocks

type canReceiveProblemsUseCase interface {
	Handle(ctx context.Context, req canreceiveproblems.Request) (canreceiveproblems.Response, error)
}

type freeHandsUseCase interface {
	Handle(ctx context.Context, req freehands.Request) error
}

//go:generate options-gen -out-filename=handler_options.gen.go -from-struct=Options
type Options struct {
	canReceiveProblemUseCase canReceiveProblemsUseCase `option:"mandatory" validate:"required"`
	freeHandsUseCase         freeHandsUseCase          `option:"mandatory" validate:"required"`
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
