package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/server"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	"github.com/zestagio/chat-service/internal/server/errhandler"
	managerload "github.com/zestagio/chat-service/internal/services/manager-load"
	inmemmanagerpool "github.com/zestagio/chat-service/internal/services/manager-pool/in-mem"
	canreceiveproblems "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems"
	freehands "github.com/zestagio/chat-service/internal/usecases/manager/free-hands"
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

	managerLoadSrv *managerload.Service,
) (*server.Server, error) {
	managerPool := inmemmanagerpool.New()

	canReceiveProblemUseCase, err := canreceiveproblems.New(canreceiveproblems.NewOptions(managerLoadSrv, managerPool))
	if err != nil {
		return nil, fmt.Errorf("create canreceiveproblem usecase: %v", err)
	}

	freeHandsUseCase, err := freehands.New(freehands.NewOptions(managerLoadSrv, managerPool))
	if err != nil {
		return nil, fmt.Errorf("create freehands usecase: %v", err)
	}

	v1Handlers, err := managerv1.NewHandlers(managerv1.NewOptions(canReceiveProblemUseCase, freeHandsUseCase))
	if err != nil {
		return nil, fmt.Errorf("create v1 handlers: %v", err)
	}

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
		func(router server.EchoRouter) {
			managerv1.RegisterHandlers(router, v1Handlers)
		},
		httpErrorHandler.Handle,
	))
	if err != nil {
		return nil, fmt.Errorf("build server: %v", err)
	}

	return srv, nil
}
