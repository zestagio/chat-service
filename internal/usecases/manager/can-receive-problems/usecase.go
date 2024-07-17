package canreceiveproblems

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=canreceiveproblemsmocks

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

type managerPool interface {
	Contains(ctx context.Context, managerID types.UserID) (bool, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	mLoadSvc managerLoadService `option:"mandatory" validate:"required"`
	mPool    managerPool        `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	return UseCase{Options: opts}, opts.Validate()
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := req.Validate(); err != nil {
		return Response{}, err
	}

	alreadyInPool, err := u.mPool.Contains(ctx, req.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("manager pool service call: %v", err)
	}
	if alreadyInPool {
		return Response{Result: false}, nil
	}

	result, err := u.mLoadSvc.CanManagerTakeProblem(ctx, req.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("manager load service call: %v", err)
	}
	return Response{Result: result}, nil
}
