package store

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

//go:generate options-gen -out-filename=client_psql_options.gen.go -from-struct=PSQLOptions
type PSQLOptions struct {
	address  string `option:"mandatory" validate:"required,hostname_port"`
	username string `option:"mandatory" validate:"required"`
	password string `option:"mandatory" validate:"required"`
	database string `option:"mandatory" validate:"required"`
	debug    bool
}

func NewPSQLClient(opts PSQLOptions) (*Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	db, err := NewPgxDB(NewPgxOptions(opts.address, opts.username, opts.password, opts.database))
	if err != nil {
		return nil, fmt.Errorf("init db driver: %v", err)
	}

	clientOpts := []Option{Driver(entsql.OpenDB(dialect.Postgres, db))}
	if opts.debug {
		l := func(a ...any) {
			zap.L().Named("store").Sugar().Debug(a...)
		}
		clientOpts = append(clientOpts, Debug(), Log(l))
	}
	return NewClient(clientOpts...), nil
}

//go:generate options-gen -out-filename=client_psql_pgx_options.gen.go -from-struct=PgxOptions
type PgxOptions struct {
	address  string `option:"mandatory" validate:"required,hostname_port"`
	username string `option:"mandatory" validate:"required"`
	password string `option:"mandatory" validate:"required"`
	database string `option:"mandatory" validate:"required"`
}

func NewPgxDB(opts PgxOptions) (*sql.DB, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	dsn := (&url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(opts.username, opts.password),
		Host:   opts.address,
		Path:   opts.database,
	}).String()
	return sql.Open("pgx", dsn)
}
