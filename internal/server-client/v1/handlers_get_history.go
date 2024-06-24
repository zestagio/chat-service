package clientv1

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	gethistory "github.com/zestagio/chat-service/internal/usecases/client/get-history"
	"github.com/zestagio/chat-service/pkg/pointer"
)

func (h Handlers) PostGetHistory(eCtx echo.Context, params PostGetHistoryParams) error {
	ctx := eCtx.Request().Context()
	clientID := middlewares.MustUserID(eCtx)

	var req GetHistoryRequest
	if err := eCtx.Bind(&req); err != nil {
		return internalerrors.NewServerError(http.StatusBadRequest, "bind request error", err)
	}

	result, err := h.getHistory.Handle(ctx, gethistory.Request{
		ID:       params.XRequestID,
		ClientID: clientID,
		PageSize: pointer.Indirect(req.PageSize),
		Cursor:   pointer.Indirect(req.Cursor),
	})
	if err != nil {
		if errors.Is(err, gethistory.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "get history invalid request", err)
		}
		if errors.Is(err, gethistory.ErrInvalidCursor) {
			return internalerrors.NewServerError(http.StatusBadRequest, "get history invalid cursor", err)
		}
		return internalerrors.NewServerError(http.StatusInternalServerError, "get history unknown error", err)
	}

	return eCtx.JSON(http.StatusOK, h.response(result))
}

func (h Handlers) response(resp gethistory.Response) GetHistoryResponse {
	msgPage := MessagesPage{Messages: make([]Message, 0, len(resp.Messages)), Next: resp.NextCursor}

	for _, msg := range resp.Messages {
		msgPage.Messages = append(msgPage.Messages, Message{
			Id:         msg.ID,
			Body:       msg.Body,
			AuthorId:   pointer.PtrWithZeroAsNil(msg.AuthorID),
			CreatedAt:  msg.CreatedAt,
			IsReceived: msg.IsReceived,
			IsBlocked:  msg.IsBlocked,
			IsService:  msg.IsService,
		})
	}

	return GetHistoryResponse{Data: &msgPage}
}
