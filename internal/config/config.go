package config

import "time"

type Config struct {
	Global   GlobalConfig   `toml:"global"`
	Log      LogConfig      `toml:"log"`
	Sentry   SentryConfig   `toml:"sentry"`
	Servers  ServersConfig  `toml:"servers"`
	Stores   StoresConfig   `toml:"stores"`
	Clients  ClientsConfig  `toml:"clients"`
	Services ServicesConfig `toml:"services"`
}

type GlobalConfig struct {
	Env string `toml:"env" validate:"required,oneof=dev stage prod"`
}

func (c GlobalConfig) IsProduction() bool {
	return c.Env == "prod"
}

type LogConfig struct {
	Level string `toml:"level" validate:"required,oneof=debug info warn error"`
}

type SentryConfig struct {
	DSN string `toml:"dsn" validate:"omitempty,url"`
}

type ServersConfig struct {
	Debug   DebugServerConfig `toml:"debug"`
	Client  APIServerConfig   `toml:"client"`
	Manager APIServerConfig   `toml:"manager"`
}

type DebugServerConfig struct {
	Addr string `toml:"addr" validate:"required,hostname_port"`
}

type APIServerConfig struct {
	Addr           string               `toml:"addr" validate:"required,hostname_port"`
	AllowOrigins   []string             `toml:"allow_origins" validate:"required"`
	SecWsProtocol  string               `toml:"sec_ws_protocol" validate:"required"`
	RequiredAccess RequiredAccessConfig `toml:"required_access"`
}

type RequiredAccessConfig struct {
	Resource string `toml:"resource" validate:"required"`
	Role     string `toml:"role" validate:"required"`
}

type StoresConfig struct {
	PSQL PSQLConfig `toml:"psql"`
}

type PSQLConfig struct {
	Addr     string `toml:"addr" validate:"required,hostname_port"`
	Username string `toml:"username" validate:"required"`
	Password string `toml:"password" validate:"required"`
	Database string `toml:"database" validate:"required"`
	Debug    bool   `toml:"debug"`
}

type ClientsConfig struct {
	Keycloak KeycloakConfig `toml:"keycloak"`
}

type KeycloakConfig struct {
	BasePath     string `toml:"base_path" validate:"required,url"`
	Realm        string `toml:"realm" validate:"required"`
	ClientID     string `toml:"client_id" validate:"required"`
	ClientSecret string `toml:"client_secret" validate:"required,alphanum"`
	DebugMode    bool   `toml:"debug_mode"`
}

type ServicesConfig struct {
	AFCVerdictsProcessor AFCVerdictsProcessorConfig `toml:"afc_verdicts_processor"`
	ManagerLoad          ManagerLoadConfig          `toml:"manager_load"`
	ManagerScheduler     ManagerSchedulerConfig     `toml:"manager_scheduler"`
	MsgProducer          MsgProducerConfig          `toml:"msg_producer"`
	Outbox               OutboxConfig               `toml:"outbox"`
}

type AFCVerdictsProcessorConfig struct {
	Brokers                  []string `toml:"brokers" validate:"min=1"`
	Consumers                int      `toml:"consumers" validate:"min=1,max=32"`
	ConsumerGroup            string   `toml:"consumer_group" validate:"required"`
	BatchSize                int      `toml:"batch_size" validate:"min=1,max=1000"`
	VerdictsTopic            string   `toml:"verdicts_topic" validate:"required"`
	VerdictsDLQTopic         string   `toml:"verdicts_dlq_topic" validate:"required"`
	VerdictsSigningPublicKey string   `toml:"verdicts_signing_public_key"`
}

type ManagerLoadConfig struct {
	MaxProblemsAtSameTime int `toml:"max_problems_at_same_time" validate:"min=1,max=30"`
}

type ManagerSchedulerConfig struct {
	Period time.Duration `toml:"period" validate:"min=1s,max=1m"`
}

type MsgProducerConfig struct {
	Brokers    []string `toml:"brokers" validate:"min=1"`
	Topic      string   `toml:"topic" validate:"required"`
	BatchSize  int      `toml:"batch_size" validate:"min=1,max=1000"`
	EncryptKey string   `toml:"encrypt_key" validate:"omitempty,hexadecimal"`
}

type OutboxConfig struct {
	Workers    int           `toml:"workers" validate:"min=1,max=32"`
	IdleTime   time.Duration `toml:"idle_time" validate:"min=1s,max=10s"`
	ReserveFor time.Duration `toml:"reserve_for" validate:"min=3s,max=10m"`
}
