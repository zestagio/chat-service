package freehands

import (
	"context"
	"errors"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
)

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrManagerOverloaded = errors.New("manager overloaded")
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=freehandsmocks

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

type managerPool interface {
	Put(ctx context.Context, managerID types.UserID) error
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

func (u UseCase) Handle(ctx context.Context, req Request) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validate request: %w: %v", ErrInvalidRequest, err)
	}

	canTakeProblem, err := u.managerLoadSrv.CanManagerTakeProblem(ctx, req.ManagerID)
	if err != nil {
		return fmt.Errorf("free hands: %v", err)
	}

	if !canTakeProblem {
		return ErrManagerOverloaded
	}

	if err := u.managerPool.Put(ctx, req.ManagerID); err != nil {
		return fmt.Errorf("free hands: %v", err)
	}

	return nil
}
