package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	canreceiveproblems "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems"
	freehandssignal "github.com/zestagio/chat-service/internal/usecases/manager/free-hands-signal"
)

func (h Handlers) PostGetFreeHandsBtnAvailability(eCtx echo.Context, params PostGetFreeHandsBtnAvailabilityParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	resp, err := h.canReceiveProblems.Handle(ctx, canreceiveproblems.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	})
	if err != nil {
		return fmt.Errorf("handle `can receive problems` use case: %v", err)
	}

	return eCtx.JSON(http.StatusOK, GetFreeHandsBtnAvailabilityResponse{
		Data: &FreeHandsBtnAvailability{Available: resp.Result},
	})
}

func (h Handlers) PostFreeHands(eCtx echo.Context, params PostFreeHandsParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	if _, err := h.freeHandsSignal.Handle(ctx, freehandssignal.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	}); err != nil {
		if errors.Is(err, freehandssignal.ErrManagerOverloaded) {
			return internalerrors.NewServerError(int(ErrorCodeManagerOverloaded), "manager overloaded", err)
		}
		return fmt.Errorf("handle `free hands signal` use case %v", err)
	}

	var empty map[string]any
	return eCtx.JSON(http.StatusOK, FreeHandsResponse{Data: &empty})
}
