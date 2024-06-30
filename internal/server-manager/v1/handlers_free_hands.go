package managerv1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	canreceiveproblems "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems"
)

func (h Handlers) PostGetFreeHandsBtnAvailability(eCtx echo.Context, params PostGetFreeHandsBtnAvailabilityParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	useCaseResponse, err := h.canReceiveProblemUseCase.Handle(ctx, canreceiveproblems.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	})
	if err != nil {
		return internalerrors.NewServerError(http.StatusInternalServerError, "internal error", err)
	}

	response := GetFreeHandsBtnAvailability{
		Available: useCaseResponse.Result,
	}

	return eCtx.JSON(http.StatusOK, &GetFreeHandsBtnAvailabilityResponse{Data: &response})
}
