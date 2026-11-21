//go:build integration

package bot_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/navyansh007/kharcha-bot/internal/httpserver"
)

func TestWebhookReplayLogsOnce(t *testing.T) {
	h := newHarness(t)
	const secret = "0123456789abcdef0123456789abcdef"
	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Options{
		Logger:  slog.New(slog.DiscardHandler),
		Webhook: &httpserver.Webhook{PathToken: "pathtoken1", Secret: secret, Updates: h.bot},
	}))
	defer srv.Close()

	body, err := json.Marshal(h.message(asha, private(asha), "120 chai"))
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/webhook/pathtoken1", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	}

	if n := h.count("SELECT count(*) FROM personal_expenses"); n != 1 {
		t.Errorf("expenses = %d, want 1", n)
	}
}
