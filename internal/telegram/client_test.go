package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *HTTPClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewHTTPClient("123:secret")
	c.baseURL = srv.URL
	return c
}

func TestSendMessage(t *testing.T) {
	var path string
	var body map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	if err := c.SendMessage(context.Background(), SendMessageParams{ChatID: 7, Text: "hi"}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if path != "/bot123:secret/sendMessage" {
		t.Errorf("path = %q", path)
	}
	if body["text"] != "hi" || body["parse_mode"] != "HTML" || body["chat_id"] != float64(7) {
		t.Errorf("body = %v", body)
	}
}

func TestAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"slow down","parameters":{"retry_after":3}}`))
	})

	err := c.SendMessage(context.Background(), SendMessageParams{ChatID: 7, Text: "hi"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.Code != 429 || apiErr.RetryAfter != 3*time.Second {
		t.Errorf("error = %+v", apiErr)
	}
}

func TestTransportErrorHidesToken(t *testing.T) {
	c := NewHTTPClient("123:secret")
	c.baseURL = "http://127.0.0.1:1"

	err := c.SendMessage(context.Background(), SendMessageParams{ChatID: 7, Text: "hi"})
	if err == nil {
		t.Fatal("SendMessage() error = nil")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("error leaks the token: %v", err)
	}
}
