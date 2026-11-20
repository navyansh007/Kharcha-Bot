//go:build integration

// Package testdb starts a throwaway Postgres for integration tests.
package testdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/navyansh007/kharcha-bot/internal/store"
)

// Start runs a Postgres container, applies the migrations and returns a pool.
// The returned function releases both and must be called when the tests finish.
func Start(ctx context.Context) (*pgxpool.Pool, func(), error) {
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("kharcha"),
		postgres.WithUsername("kharcha"),
		postgres.WithPassword("kharcha"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start postgres: %w", err)
	}
	terminate := func() { _ = testcontainers.TerminateContainer(container) }

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminate()
		return nil, nil, fmt.Errorf("connection string: %w", err)
	}
	pool, err := store.Open(ctx, url, 20)
	if err != nil {
		terminate()
		return nil, nil, err
	}
	if err := store.Migrate(ctx, pool); err != nil {
		pool.Close()
		terminate()
		return nil, nil, err
	}
	return pool, func() { pool.Close(); terminate() }, nil
}

// Reset empties every table so that a test starts from a clean database.
func Reset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	const query = `
DO $$
DECLARE name TEXT;
BEGIN
	FOR name IN
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'
	LOOP
		EXECUTE format('TRUNCATE %I RESTART IDENTITY CASCADE', name);
	END LOOP;
END $$;`
	if _, err := pool.Exec(context.Background(), query); err != nil {
		t.Fatalf("reset database: %v", err)
	}
}
