// Package config loads and validates the bot configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// Supported values of BOT_MODE.
const (
	ModePolling = "polling"
	ModeWebhook = "webhook"
)

const minSecretLength = 32

var (
	pathTokenPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{8,}$`)
	webhookSecretPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
)

// Config holds every setting the bot reads at startup.
type Config struct {
	Mode             string
	BotToken         string
	DryRun           bool
	WebhookSecret    string
	WebhookPathToken string
	PublicBaseURL    string
	DatabaseURL      string
	DBMaxConns       int32
	CronSecret       string
	UPILinkSecret    string
	Port             int
	LogLevel         slog.Level
	Location         *time.Location
	MaxExpensePaise  int64
	MetricsEnabled   bool
}

// Load builds a Config using getenv, and reports every invalid setting at once.
func Load(getenv func(string) string) (Config, error) {
	r := &reader{getenv: getenv}
	cfg := Config{
		Mode:             r.str("BOT_MODE", ModePolling),
		BotToken:         r.str("TELEGRAM_BOT_TOKEN", ""),
		DryRun:           r.boolean("TELEGRAM_DRY_RUN", false),
		WebhookSecret:    r.str("WEBHOOK_SECRET", ""),
		WebhookPathToken: r.str("WEBHOOK_PATH_TOKEN", ""),
		PublicBaseURL:    r.str("PUBLIC_BASE_URL", ""),
		DatabaseURL:      r.required("DATABASE_URL"),
		DBMaxConns:       int32(r.integer("DB_MAX_CONNS", 5, 1, 100)),
		CronSecret:       r.str("CRON_SECRET", ""),
		UPILinkSecret:    r.str("UPI_LINK_SECRET", ""),
		Port:             int(r.integer("PORT", 8080, 1, 65535)),
		MaxExpensePaise:  r.integer("MAX_EXPENSE_PAISE", 10_000_000, 100, 1_000_000_000_000),
		MetricsEnabled:   r.boolean("METRICS_ENABLED", false),
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(r.str("LOG_LEVEL", "info"))); err != nil {
		r.fail("LOG_LEVEL: %v", err)
	}
	loc, err := time.LoadLocation(r.str("TZ_DISPLAY", "Asia/Kolkata"))
	if err != nil {
		r.fail("TZ_DISPLAY: %v", err)
	}
	cfg.Location = loc

	cfg.validate(r)
	if err := errors.Join(r.errs...); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func (c Config) validate(r *reader) {
	if c.Mode != ModePolling && c.Mode != ModeWebhook {
		r.fail("BOT_MODE must be %q or %q", ModePolling, ModeWebhook)
	}
	if c.BotToken == "" && !c.DryRun {
		r.fail("TELEGRAM_BOT_TOKEN is required unless TELEGRAM_DRY_RUN is true")
	}
	if c.PublicBaseURL != "" {
		u, err := url.Parse(c.PublicBaseURL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			r.fail("PUBLIC_BASE_URL must be an https URL")
		}
	}
	for name, value := range map[string]string{
		"CRON_SECRET":     c.CronSecret,
		"UPI_LINK_SECRET": c.UPILinkSecret,
		"WEBHOOK_SECRET":  c.WebhookSecret,
	} {
		if value != "" && len(value) < minSecretLength {
			r.fail("%s must be at least %d characters", name, minSecretLength)
		}
	}
	if c.WebhookSecret != "" && !webhookSecretPattern.MatchString(c.WebhookSecret) {
		r.fail("WEBHOOK_SECRET may only contain letters, digits, _ and -, up to 256 characters")
	}
	if c.Mode == ModeWebhook {
		if c.WebhookSecret == "" {
			r.fail("WEBHOOK_SECRET is required in webhook mode")
		}
		if !pathTokenPattern.MatchString(c.WebhookPathToken) {
			r.fail("WEBHOOK_PATH_TOKEN must be 8 or more letters, digits, _ or -")
		}
	}
}

type reader struct {
	getenv func(string) string
	errs   []error
}

func (r *reader) fail(format string, args ...any) {
	r.errs = append(r.errs, fmt.Errorf(format, args...))
}

func (r *reader) str(key, fallback string) string {
	if v := r.getenv(key); v != "" {
		return v
	}
	return fallback
}

func (r *reader) required(key string) string {
	v := r.getenv(key)
	if v == "" {
		r.fail("%s is required", key)
	}
	return v
}

func (r *reader) boolean(key string, fallback bool) bool {
	v := r.getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		r.fail("%s must be true or false", key)
	}
	return b
}

func (r *reader) integer(key string, fallback, minimum, maximum int64) int64 {
	v := r.getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < minimum || n > maximum {
		r.fail("%s must be a number between %d and %d", key, minimum, maximum)
		return fallback
	}
	return n
}
