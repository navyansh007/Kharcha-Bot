// Package upi builds UPI payment links and the signed tokens that protect the
// redirect page.
package upi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/navyansh007/kharcha-bot/internal/money"
)

// TokenLifetime is how long a payment link stays valid.
const TokenLifetime = 72 * time.Hour

// Errors returned by Verify.
var (
	ErrInvalidToken = errors.New("invalid payment token")
	ErrExpiredToken = errors.New("expired payment token")
)

// Payment is the content of a UPI payment request.
type Payment struct {
	VPA    string      `json:"vpa"`
	Name   string      `json:"name"`
	Amount money.Paise `json:"amount"`
	Note   string      `json:"note"`
}

// URI returns the upi://pay link that opens a UPI app.
func (p Payment) URI() string {
	params := []struct{ key, value string }{
		{"pa", p.VPA},
		{"pn", p.Name},
		{"am", p.Amount.Rupees()},
		{"cu", "INR"},
		{"tn", p.Note},
	}
	parts := make([]string, len(params))
	for i, param := range params {
		escaped := strings.ReplaceAll(url.QueryEscape(param.value), "+", "%20")
		parts[i] = param.key + "=" + strings.ReplaceAll(escaped, "%40", "@")
	}
	return "upi://pay?" + strings.Join(parts, "&")
}

type claims struct {
	Payment
	Expires int64 `json:"exp"`
}

// Signer creates and checks payment tokens.
type Signer struct {
	key     []byte
	baseURL string
}

// NewSigner returns a Signer whose links point at baseURL.
func NewSigner(secret, baseURL string) *Signer {
	return &Signer{key: []byte(secret), baseURL: strings.TrimSuffix(baseURL, "/")}
}

// Link returns the HTTPS address of the redirect page for a payment.
func (s *Signer) Link(p Payment, now time.Time) (string, error) {
	token, err := s.sign(p, now.Add(TokenLifetime))
	if err != nil {
		return "", err
	}
	return s.baseURL + "/pay/" + token, nil
}

func (s *Signer) sign(p Payment, expires time.Time) (string, error) {
	body, err := json.Marshal(claims{Payment: p, Expires: expires.Unix()})
	if err != nil {
		return "", fmt.Errorf("encode payment: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	return payload + "." + base64.RawURLEncoding.EncodeToString(s.mac(payload)), nil
}

// Verify returns the payment in token if it is authentic and has not expired.
func (s *Signer) Verify(token string, now time.Time) (Payment, error) {
	payload, signature, ok := strings.Cut(token, ".")
	if !ok {
		return Payment{}, ErrInvalidToken
	}
	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(got, s.mac(payload)) {
		return Payment{}, ErrInvalidToken
	}
	body, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Payment{}, ErrInvalidToken
	}
	var c claims
	if err := json.Unmarshal(body, &c); err != nil {
		return Payment{}, ErrInvalidToken
	}
	if now.Unix() > c.Expires {
		return Payment{}, ErrExpiredToken
	}
	return c.Payment, nil
}

func (s *Signer) mac(payload string) []byte {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(payload))
	return h.Sum(nil)
}
