package config

type Config struct {
	Global  GlobalConfig  `toml:"global"`
	Log     LogConfig     `toml:"log"`
	Sentry  SentryConfig  `toml:"sentry"`
	Servers ServersConfig `toml:"servers"`
	Clients ClientsConfig `toml:"clients"`
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
	Dsn string `toml:"dsn" validate:"omitempty,http_url"`
}

type ServersConfig struct {
	Debug  DebugServerConfig  `toml:"debug"`
	Client ClientServerConfig `toml:"client"`
}

type DebugServerConfig struct {
	Addr string `toml:"addr" validate:"required,hostname_port"`
}

type ClientServerConfig struct {
	Addr           string                           `toml:"addr" validate:"required,hostname_port"`
	AllowOrigins   []string                         `toml:"allow_origins" validate:"required,dive,min=1,http_url"`
	RequiredAccess ClientServerRequiredAccessConfig `toml:"required_access"`
}

type ClientServerRequiredAccessConfig struct {
	Resource string `toml:"resource" validate:"required"`
	Role     string `toml:"role" validate:"required"`
}

type ClientsConfig struct {
	Keycloak KeycloakClientsConfig `toml:"keycloak"`
}

type KeycloakClientsConfig struct {
	BasePath     string `toml:"base_path" validate:"required,http_url"`
	Realm        string `toml:"realm" validate:"required"`
	ClientID     string `toml:"client_id" validate:"required"`
	ClientSecret string `toml:"client_secret" validate:"required"`
	DebugMode    bool   `toml:"debug_mode"`
}
