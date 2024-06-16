package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/config"
	"github.com/zestagio/chat-service/internal/logger"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
	serverdebug "github.com/zestagio/chat-service/internal/server-debug"
)

var configPath = flag.String("config", "configs/config.toml", "Path to config file")

func main() {
	if err := run(); err != nil {
		log.Fatalf("run app: %v", err)
	}
}

func run() (errReturned error) {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.ParseAndValidate(*configPath)
	if err != nil {
		return fmt.Errorf("parse and validate config %q: %v", *configPath, err)
	}

	logger.MustInit(
		logger.NewOptions(
			cfg.Log.Level,
			logger.WithSentryEnv(cfg.Global.Env),
			logger.WithSentryDsn(cfg.Sentry.Dsn),
			logger.WithProductionMode(cfg.Global.IsProduction()),
		),
	)
	defer logger.Sync()

	kc, err := keycloakclient.New(keycloakclient.NewOptions(
		cfg.Clients.Keycloak.BasePath,
		cfg.Clients.Keycloak.Realm,
		cfg.Clients.Keycloak.ClientID,
		cfg.Clients.Keycloak.ClientSecret,
		keycloakclient.WithDebugMode(cfg.Clients.Keycloak.DebugMode),
	))
	if err != nil {
		return fmt.Errorf("create keycloak client: %v", err)
	}
	if cfg.Global.IsProduction() && cfg.Clients.Keycloak.DebugMode {
		zap.L().Warn("keycloak client in the debug mode")
	}

	clientV1Swagger, err := clientv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("get client v1 swagger: %v", err)
	}

	srvClient, err := initServerClient(
		cfg.Servers.Client.Addr,
		cfg.Servers.Client.AllowOrigins,
		clientV1Swagger,
		kc,
		cfg.Servers.Client.RequiredAccess.Resource,
		cfg.Servers.Client.RequiredAccess.Role,
	)
	if err != nil {
		return fmt.Errorf("init client server: %v", err)
	}

	srvDebug, err := serverdebug.New(serverdebug.NewOptions(cfg.Servers.Debug.Addr, clientV1Swagger))
	if err != nil {
		return fmt.Errorf("init debug server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Run servers.
	eg.Go(func() error { return srvClient.Run(ctx) })
	eg.Go(func() error { return srvDebug.Run(ctx) })

	// Run services.
	// Ждут своего часа.
	// ...

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("wait app stop: %v", err)
	}

	return nil
}
