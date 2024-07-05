package keycloakclient

import (
	"fmt"

	"github.com/go-resty/resty/v2"

	"github.com/zestagio/chat-service/internal/buildinfo"
)

//go:generate options-gen -out-filename=client_options.gen.go -from-struct=Options
type Options struct {
	basePath     string `option:"mandatory" validate:"required,url"`
	realm        string `option:"mandatory" validate:"required"`
	clientID     string `option:"mandatory" validate:"required"`
	clientSecret string `option:"mandatory" validate:"required"`
	debugMode    bool
}

// Client is a tiny client to the Keycloak realm operations. UMA configuration:
// http://localhost:3010/realms/Bank/.well-known/uma2-configuration
type Client struct {
	realm        string
	clientID     string
	clientSecret string

	cli *resty.Client
}

func New(opts Options) (*Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	cli := resty.New()
	cli.SetDebug(opts.debugMode)
	cli.SetBaseURL(opts.basePath)
	cli.SetHeader("User-Agent", "chat-service/"+buildinfo.Version())

	return &Client{
		realm:        opts.realm,
		clientID:     opts.clientID,
		clientSecret: opts.clientSecret,
		cli:          cli,
	}, nil
}
