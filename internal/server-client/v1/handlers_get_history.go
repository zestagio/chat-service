package clientv1

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/zestagio/chat-service/internal/types"
)

var stub = MessagesPage{Messages: []Message{
	{
		AuthorId:  types.NewUserID(),
		Body:      "Здравствуйте! Разберёмся.",
		CreatedAt: time.Now(),
		Id:        types.NewMessageID(),
	},
	{
		AuthorId:  types.MustParse[types.UserID]("7ea8cd64-df5c-497b-a29e-f82c537260f9"),
		Body:      "Привет! Не могу снять денег с карты,\nпишет 'карта заблокирована'",
		CreatedAt: time.Now().Add(-time.Minute),
		Id:        types.NewMessageID(),
	},
}}

func (h Handlers) PostGetHistory(eCtx echo.Context, _ PostGetHistoryParams) error {
	return eCtx.JSON(http.StatusOK, GetHistoryResponse{Data: stub})
}
