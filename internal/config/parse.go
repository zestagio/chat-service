package config

import (
	"github.com/BurntSushi/toml"

	"github.com/zestagio/chat-service/internal/validator"
)

func ParseAndValidate(filename string) (Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		return conf, err
	}

	if err := validator.Validator.Struct(conf); err != nil {
		return conf, err
	}

	return conf, nil
}
