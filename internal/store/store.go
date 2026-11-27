// Package store connects to Postgres, runs migrations and manages transactions.
package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/navyansh007/kharcha-bot/internal/store/db"
)

const maxIdleTime = 2 * time.Minute

//go:embed migrations/*.sql
var migrations embed.FS

// Open creates a connection pool and checks that the database is reachable.
func Open(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MaxConnIdleTime = maxIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate applies all pending migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) (err error) {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	conn := stdlib.OpenDBFromPool(pool)
	defer func() { err = errors.Join(err, conn.Close()) }()

	provider, err := goose.NewProvider(goose.DialectPostgres, conn, files)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// InTx runs fn in a transaction, committing if it returns nil and rolling back otherwise.
func InTx(ctx context.Context, pool *pgxpool.Pool, fn func(*db.Queries) error) error {
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return fn(db.New(tx))
	})
	if err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	return nil
}
