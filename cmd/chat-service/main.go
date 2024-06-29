package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/config"
	"github.com/zestagio/chat-service/internal/logger"
	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	jobsrepo "github.com/zestagio/chat-service/internal/repositories/jobs"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
	serverdebug "github.com/zestagio/chat-service/internal/server-debug"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	msgproducer "github.com/zestagio/chat-service/internal/services/msg-producer"
	"github.com/zestagio/chat-service/internal/services/outbox"
	sendclientmessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-client-message"
	"github.com/zestagio/chat-service/internal/store"
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

	lg := zap.L().Named("main")

	if cfg.Global.IsProduction() && cfg.Stores.PSQL.Debug {
		lg.Warn("psql client in the debug mode")
	}

	storage, err := store.NewPSQLClient(store.NewPSQLOptions(
		cfg.Stores.PSQL.Addr,
		cfg.Stores.PSQL.Username,
		cfg.Stores.PSQL.Password,
		cfg.Stores.PSQL.Database,
		store.WithDebug(cfg.Stores.PSQL.Debug),
	))
	if err != nil {
		return fmt.Errorf("create store client: %v", err)
	}
	defer multierr.AppendInvoke(&errReturned, multierr.Close(storage))

	// Migrations.
	if err := storage.Schema.Create(ctx); err != nil {
		return fmt.Errorf("migrate: %v", err)
	}

	// Repositories.
	db := store.NewDatabase(storage)

	chatsRepo, err := chatsrepo.New(chatsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create chats repo: %v", err)
	}

	msgRepo, err := messagesrepo.New(messagesrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create messages repo: %v", err)
	}

	problemsRepo, err := problemsrepo.New(problemsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create problems repo: %v", err)
	}

	jobsRepo, err := jobsrepo.New(jobsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("init jobs repo err: %v", err)
	}

	msgProducer, err := msgproducer.New(msgproducer.NewOptions(
		msgproducer.NewKafkaWriter(
			cfg.Services.MsgProducer.Brokers,
			cfg.Services.MsgProducer.Topic,
			cfg.Services.MsgProducer.BatchSize,
		),
		msgproducer.WithEncryptKey(cfg.Services.MsgProducer.EncryptKey),
	))
	if err != nil {
		return fmt.Errorf("init msg producer err: %v", err)
	}

	outboxSrv, err := outbox.New(outbox.NewOptions(
		cfg.Services.Outbox.Workers,
		cfg.Services.Outbox.IDLE,
		cfg.Services.Outbox.ReserveFor,
		jobsRepo,
		db,
	))
	if err != nil {
		return fmt.Errorf("init outbox service err: %v", err)
	}

	sendClientMessageJob, err := sendclientmessagejob.New(sendclientmessagejob.NewOptions(msgProducer, msgRepo))
	if err != nil {
		return fmt.Errorf("create send client message job: %v", err)
	}

	if err := outboxSrv.RegisterJob(sendClientMessageJob); err != nil {
		return fmt.Errorf("register send client message job err: %v", err)
	}

	// Clients.
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

	// Servers.
	clientV1Swagger, err := clientv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("get client v1 swagger: %v", err)
	}

	srvClient, err := initServerClient(
		cfg.Global.IsProduction(),
		cfg.Servers.Client.Addr,
		cfg.Servers.Client.AllowOrigins,
		clientV1Swagger,
		kc,
		cfg.Servers.Client.RequiredAccess.Resource,
		cfg.Servers.Client.RequiredAccess.Role,
		db,
		chatsRepo,
		msgRepo,
		problemsRepo,
		outboxSrv,
	)
	if err != nil {
		return fmt.Errorf("init client server: %v", err)
	}

	managerV1Swagger, err := managerv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("get manager v1 swagger: %v", err)
	}

	srvManager, err := initServerManager(
		cfg.Global.IsProduction(),
		cfg.Servers.Manager.Addr,
		cfg.Servers.Manager.AllowOrigins,
		managerV1Swagger,
		kc,
		cfg.Servers.Manager.RequiredAccess.Resource,
		cfg.Servers.Manager.RequiredAccess.Role,
	)
	if err != nil {
		return fmt.Errorf("init manager server: %v", err)
	}

	srvDebug, err := serverdebug.New(serverdebug.NewOptions(
		cfg.Servers.Debug.Addr,
		clientV1Swagger,
		managerV1Swagger,
	))
	if err != nil {
		return fmt.Errorf("init debug server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Run servers.
	eg.Go(func() error { return srvClient.Run(ctx) })
	eg.Go(func() error { return srvManager.Run(ctx) })
	eg.Go(func() error { return srvDebug.Run(ctx) })
	eg.Go(func() error { return outboxSrv.Run(ctx) })

	// Run services.
	// Ждут своего часа.
	// ...

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("wait app stop: %v", err)
	}

	return nil
}
