package middlewares

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/introspector_mock.gen.go -package=middlewaresmocks Introspector

const tokenCtxKey = "user-token"

var ErrNoRequiredResourceRole = errors.New("no required resource role")

type Introspector interface {
	IntrospectToken(ctx context.Context, token string) (*keycloakclient.IntrospectTokenResult, error)
}

// NewKeycloakTokenAuth returns a middleware that implements "active" authentication:
// each request is verified by the Keycloak server.
func NewKeycloakTokenAuth(introspector Introspector, resource, role string) echo.MiddlewareFunc {
	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup:  "header:" + echo.HeaderAuthorization,
		AuthScheme: "Bearer",
		Validator: func(tokenStr string, eCtx echo.Context) (bool, error) {
			res, err := introspector.IntrospectToken(context.Background(), tokenStr)
			if err != nil {
				return false, err
			}
			if !res.Active {
				return false, nil
			}

			var cl claims
			t, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &cl)
			if err != nil {
				// Unreachable.
				return false, err
			}
			if err := t.Claims.Valid(); err != nil {
				return false, err
			}
			if !cl.ResourcesAccess.HasResourceRole(resource, role) {
				return false, echo.ErrForbidden.WithInternal(ErrNoRequiredResourceRole)
			}

			eCtx.Set(tokenCtxKey, t)
			return true, nil
		},
	})
}

func MustUserID(eCtx echo.Context) types.UserID {
	uid, ok := userID(eCtx)
	if !ok {
		panic("no user token in request context")
	}
	return uid
}

func userID(eCtx echo.Context) (types.UserID, bool) {
	t := eCtx.Get(tokenCtxKey)
	if t == nil {
		return types.UserIDNil, false
	}

	tt, ok := t.(*jwt.Token)
	if !ok {
		return types.UserIDNil, false
	}

	userIDProvider, ok := tt.Claims.(interface{ UserID() types.UserID })
	if !ok {
		return types.UserIDNil, false
	}
	return userIDProvider.UserID(), true
}
