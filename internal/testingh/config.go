//go:build integration

package testingh

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"

	"github.com/zestagio/chat-service/internal/logger"
	"github.com/zestagio/chat-service/internal/validator"
)

var Config config

type config struct {
	LogLevel string `envconfig:"LOG_LEVEL" default:"info" validate:"required,oneof=debug info warn error"`

	PostgresAddress  string `envconfig:"PSQL_ADDRESS" default:"localhost:5432" validate:"required,hostname_port"`
	PostgresUser     string `envconfig:"PSQL_USER" validate:"required"`
	PostgresPassword string `envconfig:"PSQL_PASSWORD" validate:"required"`
	PostgresDebug    bool   `envconfig:"PSQL_DEBUG" default:"false"`

	KeycloakBasePath     string `envconfig:"KEYCLOAK_BASE_PATH" default:"http://localhost:3010" validate:"required,url"`
	KeycloakRealm        string `envconfig:"KEYCLOAK_REALM" default:"Testing" validate:"required"`
	KeycloakClientID     string `envconfig:"KEYCLOAK_CLIENT_ID" validate:"required"`
	KeycloakClientSecret string `envconfig:"KEYCLOAK_CLIENT_SECRET" validate:"required,alphanum"`
	KeycloakTestUser     string `envconfig:"KEYCLOAK_TEST_USER" validate:"required"`
	KeycloakTestPassword string `envconfig:"KEYCLOAK_TEST_PASSWORD" validate:"required"`
}

func init() {
	if err := envconfig.Process("TEST", &Config); err != nil {
		panic(fmt.Sprintf("parse testing config: %v", err))
	}

	if err := validator.Validator.Struct(Config); err != nil {
		panic(fmt.Sprintf("validate testing config: %v", err))
	}

	logger.MustInit(logger.NewOptions(Config.LogLevel))
}
