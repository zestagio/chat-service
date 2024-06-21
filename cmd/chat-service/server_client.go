package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	serverclient "github.com/zestagio/chat-service/internal/server-client"
	"github.com/zestagio/chat-service/internal/server-client/errhandler"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
	"github.com/zestagio/chat-service/internal/store"
	gethistory "github.com/zestagio/chat-service/internal/usecases/client/get-history"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/client/send-message"
)

const nameServerClient = "server-client"

func initServerClient(
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,

	keycloak *keycloakclient.Client,
	requiredResource string,
	requiredRole string,

	msgRepo *messagesrepo.Repo,
	chatRepo *chatsrepo.Repo,
	problemRepo *problemsrepo.Repo,

	db *store.Database,

	productionMode bool,
) (*serverclient.Server, error) {
	lg := zap.L().Named(nameServerClient)

	getHistoryUseCase, err := gethistory.New(gethistory.NewOptions(msgRepo))
	if err != nil {
		return nil, fmt.Errorf("create get history usecase: %v", err)
	}

	sendMsgUseCase, err := sendmessage.New(sendmessage.NewOptions(chatRepo, msgRepo, problemRepo, db))
	if err != nil {
		return nil, fmt.Errorf("create send message usecase: %v", err)
	}

	v1Handlers, err := clientv1.NewHandlers(clientv1.NewOptions(lg, getHistoryUseCase, sendMsgUseCase))
	if err != nil {
		return nil, fmt.Errorf("create v1 handlers: %v", err)
	}

	errHandler, err := errhandler.New(errhandler.NewOptions(lg, productionMode, errhandler.ResponseBuilder))
	if err != nil {
		return nil, fmt.Errorf("create err handler: %v", err)
	}

	srv, err := serverclient.New(serverclient.NewOptions(
		lg,
		addr,
		allowOrigins,
		keycloak,
		requiredResource,
		requiredRole,
		v1Swagger,
		v1Handlers,
		errHandler.Handle,
	))
	if err != nil {
		return nil, fmt.Errorf("build server: %v", err)
	}

	return srv, nil
}
