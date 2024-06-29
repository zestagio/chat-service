package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/server"
	"github.com/zestagio/chat-service/internal/server/errhandler"
)

const nameServerManager = "server-manager"

func initServerManager(
	productionMode bool,
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,

	keycloak *keycloakclient.Client,
	requiredResource string,
	requiredRole string,
) (*server.Server, error) {
	lg := zap.L().Named(nameServerManager)

	httpErrorHandler, err := errhandler.New(errhandler.NewOptions(
		lg,
		productionMode,
		errhandler.ResponseBuilder,
	))
	if err != nil {
		return nil, fmt.Errorf("create http error handler: %v", err)
	}

	srv, err := server.New(server.NewOptions(
		lg,
		addr,
		allowOrigins,
		keycloak,
		requiredResource,
		requiredRole,
		v1Swagger,
		func(_ server.EchoRouter) {},
		httpErrorHandler.Handle,
	))
	if err != nil {
		return nil, fmt.Errorf("build server: %v", err)
	}

	return srv, nil
}
