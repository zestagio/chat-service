package serverclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomdlwr "github.com/labstack/echo/v4/middleware"
	oapimdlwr "github.com/oapi-codegen/echo-middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/zestagio/chat-service/internal/middlewares"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
)

const (
	bodyLimit = "12KB" // ~ 3000 characters * 4 bytes.

	readHeaderTimeout = time.Second
	shutdownTimeout   = 3 * time.Second
)

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options
type Options struct {
	logger           *zap.Logger              `option:"mandatory" validate:"required"`
	addr             string                   `option:"mandatory" validate:"required,hostname_port"`
	allowOrigins     []string                 `option:"mandatory" validate:"min=1"`
	introspector     middlewares.Introspector `option:"mandatory" validate:"required"`
	requiredResource string                   `option:"mandatory" validate:"required"`
	requiredRole     string                   `option:"mandatory" validate:"required"`
	v1Swagger        *openapi3.T              `option:"mandatory" validate:"required"`
	v1Handlers       clientv1.ServerInterface `option:"mandatory" validate:"required"`
	errorHandler     echo.HTTPErrorHandler    `option:"mandatory" validate:"required"`
}

type Server struct {
	lg  *zap.Logger
	srv *http.Server
}

func New(opts Options) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	e := echo.New()
	e.HTTPErrorHandler = opts.errorHandler

	e.Use(
		middlewares.NewRequestLogger(opts.logger),
		middlewares.NewRecovery(opts.logger),
		echomdlwr.CORSWithConfig(echomdlwr.CORSConfig{
			AllowOrigins: opts.allowOrigins,
			AllowMethods: []string{http.MethodPost},
		}),
		middlewares.NewKeycloakTokenAuth(opts.introspector, opts.requiredResource, opts.requiredRole),
		echomdlwr.BodyLimit(bodyLimit),
	)

	v1 := e.Group("v1", oapimdlwr.OapiRequestValidatorWithOptions(opts.v1Swagger, &oapimdlwr.Options{
		Options: openapi3filter.Options{
			ExcludeRequestBody:  false,
			ExcludeResponseBody: true,
			AuthenticationFunc:  openapi3filter.NoopAuthenticationFunc,
		},
	}))
	clientv1.RegisterHandlers(v1, opts.v1Handlers)

	return &Server{
		lg: opts.logger,
		srv: &http.Server{
			Addr:              opts.addr,
			Handler:           e,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return s.srv.Shutdown(ctx) //nolint:contextcheck // graceful shutdown with new context
	})

	eg.Go(func() error {
		s.lg.Info("listen and serve", zap.String("addr", s.srv.Addr))

		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %v", err)
		}
		return nil
	})

	return eg.Wait()
}
