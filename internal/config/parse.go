package config

import (
	"fmt"

	"github.com/BurntSushi/toml"

	"github.com/zestagio/chat-service/internal/validator"
)

func ParseAndValidate(filename string) (Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		return Config{}, fmt.Errorf("decode file: %v", err)
	}

	if err := validator.Validator.Struct(conf); err != nil {
		return Config{}, fmt.Errorf("validate: %v", err)
	}

	return conf, nil
}
