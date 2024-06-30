package config

import "time"

type Config struct {
	Global   GlobalConfig   `toml:"global"`
	Log      LogConfig      `toml:"log"`
	Sentry   SentryConfig   `toml:"sentry"`
	Servers  ServersConfig  `toml:"servers"`
	Services ServicesConfig `toml:"services"`
	Stores   StoresConfig   `toml:"stores"`
	Clients  ClientsConfig  `toml:"clients"`
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
	Dsn string `toml:"dsn" validate:"omitempty,url"`
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
	RequiredAccess RequiredAccessConfig `toml:"required_access"`
}

type RequiredAccessConfig struct {
	Resource string `toml:"resource" validate:"required"`
	Role     string `toml:"role" validate:"required"`
}

type ServicesConfig struct {
	Outbox      OutboxConfig      `toml:"outbox"`
	MsgProducer MsgProducerConfig `toml:"msg_producer"`
	ManagerLoad ManagerLoadConfig `toml:"manager_load"`
}

type OutboxConfig struct {
	Workers    int           `toml:"workers" validate:"required,min=1"`
	IDLE       time.Duration `toml:"idle_time" validate:"required,min=500ms,max=10s"`
	ReserveFor time.Duration `toml:"reserve_for" validate:"required"`
}

type MsgProducerConfig struct {
	Brokers    []string `toml:"brokers" validate:"required,gt=0,dive,required,hostname_port"`
	Topic      string   `toml:"topic" validate:"required"`
	BatchSize  int      `toml:"batch_size" validate:"required,min=1"`
	EncryptKey string   `toml:"encrypt_key" validate:"omitempty,hexadecimal"`
}

type ManagerLoadConfig struct {
	MaxProblemsAtSameTime int `toml:"max_problems_at_same_time" validate:"required,gt=0"`
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
