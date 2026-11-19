package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	secret := strings.Repeat("s", 32)
	base := map[string]string{
		"TELEGRAM_BOT_TOKEN": "123:abc",
		"DATABASE_URL":       "postgres://localhost/kharcha",
	}

	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{name: "defaults"},
		{name: "missing database", env: map[string]string{"DATABASE_URL": ""}, wantErr: "DATABASE_URL"},
		{name: "unknown mode", env: map[string]string{"BOT_MODE": "push"}, wantErr: "BOT_MODE"},
		{name: "short secret", env: map[string]string{"CRON_SECRET": "short"}, wantErr: "CRON_SECRET"},
		{name: "http base url", env: map[string]string{"PUBLIC_BASE_URL": "http://x.test"}, wantErr: "PUBLIC_BASE_URL"},
		{name: "webhook without secret", env: map[string]string{"BOT_MODE": "webhook"}, wantErr: "WEBHOOK_SECRET"},
		{
			name: "webhook complete",
			env:  map[string]string{"BOT_MODE": "webhook", "WEBHOOK_SECRET": secret, "WEBHOOK_PATH_TOKEN": "pathtoken1"},
		},
		{name: "bad port", env: map[string]string{"PORT": "0"}, wantErr: "PORT"},
		{name: "dry run needs no token", env: map[string]string{"TELEGRAM_BOT_TOKEN": "", "TELEGRAM_DRY_RUN": "true"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				if v, ok := tt.env[key]; ok {
					return v
				}
				return base[key]
			}
			_, err := Load(getenv)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Load() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want it to mention %s", err, tt.wantErr)
			}
		})
	}
}
