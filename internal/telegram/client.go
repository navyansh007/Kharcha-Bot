// Package telegram is a small client for the parts of the Bot API the bot uses.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL   = "https://api.telegram.org"
	requestTimeout   = 15 * time.Second
	maxResponseBytes = 10 << 20
	parseMode        = "HTML"
)

// Client is the set of Bot API methods the bot depends on.
type Client interface {
	GetMe(ctx context.Context) (User, error)
	GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error)
	SendMessage(ctx context.Context, params SendMessageParams) error
	EditMessageText(ctx context.Context, params EditMessageTextParams) error
	AnswerCallbackQuery(ctx context.Context, params AnswerCallbackQueryParams) error
	SendDocument(ctx context.Context, doc Document) error
}

// APIError is an error response from the Bot API.
type APIError struct {
	Method      string
	Code        int
	Description string
	RetryAfter  time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram %s: %d %s", e.Method, e.Code, e.Description)
}

// HTTPClient calls the Bot API over HTTP.
type HTTPClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewHTTPClient returns a client that authenticates with token.
func NewHTTPClient(token string) *HTTPClient {
	return &HTTPClient{baseURL: defaultBaseURL, token: token, http: &http.Client{}}
}

// GetMe returns the bot's own account.
func (c *HTTPClient) GetMe(ctx context.Context) (User, error) {
	var me User
	err := c.callJSON(ctx, requestTimeout, "getMe", struct{}{}, &me)
	return me, err
}

// GetUpdates long-polls for updates with an ID of at least offset.
func (c *HTTPClient) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	params := map[string]any{
		"offset":          offset,
		"timeout":         int(timeout.Seconds()),
		"allowed_updates": []string{"message", "callback_query", "my_chat_member"},
	}
	var updates []Update
	err := c.callJSON(ctx, timeout+requestTimeout, "getUpdates", params, &updates)
	return updates, err
}

// SendMessage sends an HTML message to a chat.
func (c *HTTPClient) SendMessage(ctx context.Context, params SendMessageParams) error {
	body := struct {
		SendMessageParams
		ParseMode string `json:"parse_mode"`
	}{params, parseMode}
	return c.callJSON(ctx, requestTimeout, "sendMessage", body, nil)
}

// EditMessageText replaces the text of a message the bot sent earlier.
func (c *HTTPClient) EditMessageText(ctx context.Context, params EditMessageTextParams) error {
	body := struct {
		EditMessageTextParams
		ParseMode string `json:"parse_mode"`
	}{params, parseMode}
	return c.callJSON(ctx, requestTimeout, "editMessageText", body, nil)
}

// AnswerCallbackQuery acknowledges a button press.
func (c *HTTPClient) AnswerCallbackQuery(ctx context.Context, params AnswerCallbackQueryParams) error {
	return c.callJSON(ctx, requestTimeout, "answerCallbackQuery", params, nil)
}

// SendDocument uploads a file to a chat.
func (c *HTTPClient) SendDocument(ctx context.Context, doc Document) error {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("chat_id", strconv.FormatInt(doc.ChatID, 10)); err != nil {
		return fmt.Errorf("telegram sendDocument: %w", err)
	}
	file, err := form.CreateFormFile("document", doc.Name)
	if err != nil {
		return fmt.Errorf("telegram sendDocument: %w", err)
	}
	if _, err := file.Write(doc.Content); err != nil {
		return fmt.Errorf("telegram sendDocument: %w", err)
	}
	if err := form.Close(); err != nil {
		return fmt.Errorf("telegram sendDocument: %w", err)
	}
	return c.call(ctx, requestTimeout, "sendDocument", form.FormDataContentType(), &body, nil)
}

func (c *HTTPClient) callJSON(ctx context.Context, timeout time.Duration, method string, params, result any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("telegram %s: encode: %w", method, err)
	}
	return c.call(ctx, timeout, method, "application/json", bytes.NewReader(body), result)
}

func (c *HTTPClient) call(
	ctx context.Context,
	timeout time.Duration,
	method, contentType string,
	body io.Reader,
	result any,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	endpoint := c.baseURL + "/bot" + c.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("telegram %s: build request failed", method)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, withoutURL(err))
	}
	defer func() { _ = resp.Body.Close() }()

	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		ErrorCode   int             `json:"error_code"`
		Description string          `json:"description"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&envelope); err != nil {
		return fmt.Errorf("telegram %s: decode status %d: %w", method, resp.StatusCode, err)
	}
	if !envelope.OK {
		return &APIError{
			Method:      method,
			Code:        envelope.ErrorCode,
			Description: envelope.Description,
			RetryAfter:  time.Duration(envelope.Parameters.RetryAfter) * time.Second,
		}
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("telegram %s: decode result: %w", method, err)
	}
	return nil
}

func withoutURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
