// Command bot runs the Kharcha Telegram bot.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	_ "time/tzdata"

	"github.com/navyansh007/kharcha-bot/internal/bot"
	"github.com/navyansh007/kharcha-bot/internal/config"
	"github.com/navyansh007/kharcha-bot/internal/httpserver"
	"github.com/navyansh007/kharcha-bot/internal/money"
	"github.com/navyansh007/kharcha-bot/internal/store"
	"github.com/navyansh007/kharcha-bot/internal/telegram"
	"github.com/navyansh007/kharcha-bot/internal/upi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("bot stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := store.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	client := newTelegramClient(cfg, logger)
	me, err := client.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("identify bot: %w", err)
	}

	var signer *upi.Signer
	if cfg.UPILinkSecret != "" && cfg.PublicBaseURL != "" {
		signer = upi.NewSigner(cfg.UPILinkSecret, cfg.PublicBaseURL)
	}

	b := bot.New(bot.Options{
		Pool:        pool,
		Client:      client,
		Logger:      logger,
		Location:    cfg.Location,
		BotUsername: me.Username,
		MaxExpense:  money.Paise(cfg.MaxExpensePaise),
	})

	routes := httpserver.Options{Logger: logger, Pay: signer}
	var wg sync.WaitGroup
	defer wg.Wait()
	defer stop()

	switch cfg.Mode {
	case config.ModeWebhook:
		routes.Webhook = &httpserver.Webhook{
			PathToken: cfg.WebhookPathToken,
			Secret:    cfg.WebhookSecret,
			Updates:   b,
		}
	default:
		wg.Go(func() { b.Poll(ctx) })
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	return httpserver.Run(ctx, addr, httpserver.NewHandler(routes), logger)
}

func newTelegramClient(cfg config.Config, logger *slog.Logger) telegram.Client {
	if cfg.DryRun {
		return telegram.DryRun{Logger: logger}
	}
	return telegram.NewHTTPClient(cfg.BotToken)
}
