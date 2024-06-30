package managerv1

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	freehands "github.com/zestagio/chat-service/internal/usecases/manager/free-hands"
)

func (h Handlers) PostFreeHands(eCtx echo.Context, params PostFreeHandsParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	err := h.freeHandsUseCase.Handle(ctx, freehands.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	})
	if errors.Is(err, freehands.ErrManagerOverloaded) {
		return internalerrors.NewServerError(ErrorCodeManagerOverloaded, "manager overloaded", err)
	}

	if err != nil {
		return internalerrors.NewServerError(http.StatusInternalServerError, "internal error", err)
	}

	return eCtx.JSON(http.StatusOK, &FreeHandsResponse{})
}
