package freehandssignal

import (
	"context"
	"errors"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=freehandssignalmocks

var ErrManagerOverloaded = errors.New("manager overloaded")

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

type managerPool interface {
	Put(ctx context.Context, managerID types.UserID) error
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

	ok, err := u.mLoadSvc.CanManagerTakeProblem(ctx, req.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("manager load service call: %v", err)
	}
	if !ok {
		return Response{}, fmt.Errorf("%w: manager cannot take more problems", ErrManagerOverloaded)
	}

	if err := u.mPool.Put(ctx, req.ManagerID); err != nil {
		return Response{}, fmt.Errorf("put manager in the pool: %v", err)
	}

	return Response{}, nil
}
