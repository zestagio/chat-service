package servermanager

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	oapimdlwr "github.com/oapi-codegen/echo-middleware"

	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
)

func NewHandlersRegistrar(
	v1Swagger *openapi3.T,
	v1Handlers managerv1.ServerInterface,
	httpErrorHandler echo.HTTPErrorHandler,
	keycloakTokenAuth echo.MiddlewareFunc,
) func(e *echo.Echo) {
	return func(e *echo.Echo) {
		v1 := e.Group("v1",
			keycloakTokenAuth,
			oapimdlwr.OapiRequestValidatorWithOptions(v1Swagger, &oapimdlwr.Options{
				Options: openapi3filter.Options{
					ExcludeRequestBody:  false,
					ExcludeResponseBody: true,
					AuthenticationFunc:  openapi3filter.NoopAuthenticationFunc,
				},
			}))
		managerv1.RegisterHandlers(v1, v1Handlers)

		e.HTTPErrorHandler = httpErrorHandler
	}
}
