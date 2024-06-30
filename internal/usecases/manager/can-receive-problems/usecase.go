package canreceiveproblems

import (
	"context"
	"errors"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
)

var (
	ErrInvalidRequest      = errors.New("invalid request")
	ErrManagerPoolContains = errors.New("manager pool error")
	ErrManagerLoadService  = errors.New("manager load service error")
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
	managerLoadSrv managerLoadService `option:"mandatory" validate:"required"`
	managerPool    managerPool        `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("validate options: %v", err)
	}
	return UseCase{Options: opts}, nil
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := req.Validate(); err != nil {
		return Response{}, fmt.Errorf("validate request: %w: %v", ErrInvalidRequest, err)
	}

	inPool, err := u.managerPool.Contains(ctx, req.ManagerID)
	if err != nil {
		return Response{}, ErrManagerPoolContains
	}

	if inPool {
		return Response{Result: false}, nil
	}

	canTakeProblem, err := u.managerLoadSrv.CanManagerTakeProblem(ctx, req.ManagerID)
	if err != nil {
		return Response{}, ErrManagerLoadService
	}

	return Response{Result: canTakeProblem}, nil
}
