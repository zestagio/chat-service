package clientv1

import (
	"errors"
	"fmt"
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
		return fmt.Errorf("bind request: %w", err)
	}

	resp, err := h.getHistory.Handle(ctx, gethistory.Request{
		ID:       params.XRequestID,
		ClientID: clientID,
		Cursor:   pointer.Indirect(req.Cursor),
		PageSize: pointer.Indirect(req.PageSize),
	})
	if err != nil {
		if errors.Is(err, gethistory.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}

		if errors.Is(err, gethistory.ErrInvalidCursor) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid cursor", err)
		}

		return fmt.Errorf("handle `get history`: %v", err)
	}

	page := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		mm := Message{
			AuthorId:   m.AuthorID.AsPointer(),
			Body:       m.Body,
			CreatedAt:  m.CreatedAt,
			Id:         m.ID,
			IsBlocked:  m.IsBlocked,
			IsReceived: m.IsReceived,
			IsService:  m.IsService,
		}
		page = append(page, mm)
	}

	return eCtx.JSON(http.StatusOK, GetHistoryResponse{Data: &MessagesPage{
		Messages: page,
		Next:     resp.NextCursor,
	}})
}
