// Command migrate applies the database migrations to DATABASE_URL.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")
}

func run() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := store.Open(ctx, url, 2)
	if err != nil {
		return err
	}
	defer pool.Close()

	return store.Migrate(ctx, pool)
}
