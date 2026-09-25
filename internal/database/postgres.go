package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"kubometr/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn(cfg))
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// dsn escapes credentials so passwords with characters like "@" or "/" work.
func dsn(cfg *config.Config) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.PostgresUser, cfg.PostgresPassword),
		Host:   net.JoinHostPort(cfg.PostgresHost, strconv.Itoa(cfg.PostgresPort)),
		Path:   "/" + cfg.PostgresDB,
	}
	return u.String()
}
