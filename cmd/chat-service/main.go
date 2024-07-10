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
	clientevents "github.com/zestagio/chat-service/internal/server-client/events"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
	serverdebug "github.com/zestagio/chat-service/internal/server-debug"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	afcverdictsprocessor "github.com/zestagio/chat-service/internal/services/afc-verdicts-processor"
	inmemeventstream "github.com/zestagio/chat-service/internal/services/event-stream/in-mem"
	managerload "github.com/zestagio/chat-service/internal/services/manager-load"
	inmemmanagerpool "github.com/zestagio/chat-service/internal/services/manager-pool/in-mem"
	msgproducer "github.com/zestagio/chat-service/internal/services/msg-producer"
	"github.com/zestagio/chat-service/internal/services/outbox"
	clientmessageblockedjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/client-message-blocked"
	clientmessagesentjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/client-message-sent"
	sendclientmessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-client-message"
	"github.com/zestagio/chat-service/internal/store"
	websocketstream "github.com/zestagio/chat-service/internal/websocket-stream"
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

	if err := logger.Init(logger.NewOptions(
		cfg.Log.Level,
		logger.WithProductionMode(cfg.Global.IsProduction()),
		logger.WithSentryDSN(cfg.Sentry.DSN),
		logger.WithSentryEnv(cfg.Global.Env),
	)); err != nil {
		return fmt.Errorf("init logger: %v", err)
	}
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

	jobsRepo, err := jobsrepo.New(jobsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create jobs repo: %v", err)
	}

	msgRepo, err := messagesrepo.New(messagesrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create messages repo: %v", err)
	}

	problemsRepo, err := problemsrepo.New(problemsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create problems repo: %v", err)
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

	// Infrastructure Services.
	msgProducer, err := msgproducer.New(msgproducer.NewOptions(
		msgproducer.NewKafkaWriter(
			cfg.Services.MsgProducer.Brokers,
			cfg.Services.MsgProducer.Topic,
			cfg.Services.MsgProducer.BatchSize,
		),
		msgproducer.WithEncryptKey(cfg.Services.MsgProducer.EncryptKey),
	))
	if err != nil {
		return fmt.Errorf("create message producer: %v", err)
	}
	defer multierr.AppendInvoke(&errReturned, multierr.Close(msgProducer))

	outBox, err := outbox.New(outbox.NewOptions(
		cfg.Services.Outbox.Workers,
		cfg.Services.Outbox.IdleTime,
		cfg.Services.Outbox.ReserveFor,
		jobsRepo,
		db,
	))
	if err != nil {
		return fmt.Errorf("create outbox service: %v", err)
	}

	afcVerdictProcessor, err := afcverdictsprocessor.New(afcverdictsprocessor.NewOptions(
		cfg.Services.AFCVerdictsProcessor.Brokers,
		cfg.Services.AFCVerdictsProcessor.Consumers,
		cfg.Services.AFCVerdictsProcessor.ConsumerGroup,
		cfg.Services.AFCVerdictsProcessor.VerdictsTopic,
		afcverdictsprocessor.NewKafkaReader,
		afcverdictsprocessor.NewKafkaDLQWriter(
			cfg.Services.AFCVerdictsProcessor.Brokers,
			cfg.Services.AFCVerdictsProcessor.VerdictsDLQTopic,
		),
		db,
		msgRepo,
		outBox,
		afcverdictsprocessor.WithVerdictsSignKey(cfg.Services.AFCVerdictsProcessor.VerdictsSigningPublicKey),
	))
	if err != nil {
		return fmt.Errorf("create afc verdicts processor service: %v", err)
	}

	managerPool := inmemmanagerpool.New()
	defer multierr.AppendInvoke(&errReturned, multierr.Close(managerPool))

	// Domain Services.
	managerLoad, err := managerload.New(managerload.NewOptions(
		cfg.Services.ManagerLoad.MaxProblemsAtSameTime,
		problemsRepo,
	))
	if err != nil {
		return fmt.Errorf("create manager load service: %v", err)
	}

	eventStream := inmemeventstream.New()
	defer eventStream.Close()

	// Application Services. Jobs.
	for _, j := range []outbox.Job{
		sendclientmessagejob.Must(sendclientmessagejob.NewOptions(msgProducer, msgRepo, eventStream)),
		clientmessagesentjob.Must(clientmessagesentjob.NewOptions(msgRepo, eventStream)),
		clientmessageblockedjob.Must(clientmessageblockedjob.NewOptions(msgRepo, eventStream)),
	} {
		outBox.MustRegisterJob(j)
	}

	shutdown := make(chan struct{})

	// Websocket client stream.
	wsClient, err := websocketstream.NewHTTPHandler(websocketstream.NewOptions(
		zap.L().Named("websocket-client"),
		eventStream,
		clientevents.Adapter{},
		websocketstream.JSONEventWriter{},
		websocketstream.NewUpgrader(cfg.Servers.Client.AllowOrigins, cfg.Servers.Client.SecWSProtocol),
		shutdown,
	))
	if err != nil {
		return fmt.Errorf("websocket client stream: %v", err)
	}

	// Websocket manager stream.
	wsManager, err := websocketstream.NewHTTPHandler(websocketstream.NewOptions(
		zap.L().Named("websocket-manager"),
		eventStream,
		clientevents.Adapter{},
		websocketstream.JSONEventWriter{},
		websocketstream.NewUpgrader(cfg.Servers.Manager.AllowOrigins, cfg.Servers.Manager.SecWSProtocol),
		shutdown,
	))
	if err != nil {
		return fmt.Errorf("websocket manager stream: %v", err)
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
		outBox,
		db,
		chatsRepo,
		msgRepo,
		problemsRepo,
		wsClient,
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
		managerLoad,
		managerPool, // как sql-конфиг
		wsManager,
	)
	if err != nil {
		return fmt.Errorf("init manager server: %v", err)
	}

	eventsSwagger, err := clientevents.GetSwagger()
	if err != nil {
		return fmt.Errorf("get events swagger: %v", err)
	}

	srvDebug, err := serverdebug.New(serverdebug.NewOptions(
		cfg.Servers.Debug.Addr,
		clientV1Swagger,
		managerV1Swagger,
		eventsSwagger,
	))
	if err != nil {
		return fmt.Errorf("init debug server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Run servers.
	eg.Go(func() error { return srvClient.Run(ctx) })
	eg.Go(func() error { return srvManager.Run(ctx) })
	eg.Go(func() error { return srvDebug.Run(ctx) })

	// Run services.
	eg.Go(func() error { return outBox.Run(ctx) })
	eg.Go(func() error { return afcVerdictProcessor.Run(ctx) })

	// Websocket shutdown.
	eg.Go(func() error {
		<-ctx.Done()

		shutdown <- struct{}{}
		return nil
	})

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("wait app stop: %v", err)
	}

	return nil
}
