package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echomdlwr "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/zestagio/chat-service/internal/middlewares"
)

const (
	bodyLimit = "12KB" // ~ 3000 characters * 4 bytes.

	readHeaderTimeout = time.Second
	shutdownTimeout   = 3 * time.Second
)

type wsHTTPHandler interface {
	Serve(eCtx echo.Context) error
}

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options
type Options struct {
	logger            *zap.Logger              `option:"mandatory" validate:"required"`
	addr              string                   `option:"mandatory" validate:"required,hostname_port"`
	allowOrigins      []string                 `option:"mandatory" validate:"min=1"`
	introspector      middlewares.Introspector `option:"mandatory" validate:"required"`
	requiredResource  string                   `option:"mandatory" validate:"required"`
	requiredRole      string                   `option:"mandatory" validate:"required"`
	handlersRegistrar func(e *echo.Echo)       `option:"mandatory" validate:"required"`
	wsHandler         wsHTTPHandler            `option:"mandatory" validate:"required"`
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
	e.Use(
		middlewares.NewRequestLogger(opts.logger),
		middlewares.NewRecovery(opts.logger),
		echomdlwr.CORSWithConfig(echomdlwr.CORSConfig{
			AllowOrigins: opts.allowOrigins,
			AllowMethods: []string{http.MethodPost},
		}),
		echomdlwr.BodyLimit(bodyLimit),
	)

	e.GET(
		"/ws",
		opts.wsHandler.Serve,
		middlewares.NewKeycloakWSTokenAuth(opts.introspector, opts.requiredResource, opts.requiredRole),
	)

	opts.handlersRegistrar(e)

	srv := &http.Server{
		Addr:              opts.addr,
		Handler:           e,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return &Server{
		lg:  opts.logger,
		srv: srv,
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
