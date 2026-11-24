package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

func TestMetricsAreExported(t *testing.T) {
	m := New()
	m.ObserveUpdate("duplicate", "", time.Millisecond)
	m.ObserveUpdate("processed", "split", time.Millisecond)
	client := m.Instrument(&telegram.Fake{})
	if err := client.SendMessage(context.Background(), telegram.SendMessageParams{ChatID: 1}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	for _, want := range []string{
		`updates_total{outcome="duplicate"} 1`,
		`commands_total{name="split"} 1`,
		`telegram_send_total{status="ok"} 1`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestNilMetricsIsSafe(*testing.T) {
	var m *Metrics
	m.ObserveUpdate("processed", "split", time.Second)
	m.ObserveSend(nil)
}
