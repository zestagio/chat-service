package middlewares

import (
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"

	"github.com/zestagio/chat-service/internal/types"
)

func SetToken(c echo.Context, uid types.UserID) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{
		Subject: uid,
	})

	c.Set(tokenCtxKey, token)
}
