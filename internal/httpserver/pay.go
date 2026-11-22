package httpserver

import (
	"bytes"
	_ "embed"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/navyansh007/kharcha-bot/internal/upi"
)

//go:embed pay.html
var payHTML string

var payPage = template.Must(template.New("pay").Parse(payHTML))

func payHandler(signer *upi.Signer, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payment, err := signer.Verify(chi.URLParam(r, "token"), time.Now())
		if errors.Is(err, upi.ErrExpiredToken) {
			http.Error(w, "This payment link has expired. Send /balance for a new one.", http.StatusGone)
			return
		}
		if err != nil {
			http.NotFound(w, r)
			return
		}

		var body bytes.Buffer
		err = payPage.Execute(&body, struct {
			upi.Payment
			Link template.URL
		}{payment, template.URL(payment.URI())})
		if err != nil {
			logger.Error("render payment page", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("Cache-Control", "no-store")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("X-Robots-Tag", "noindex")
		_, _ = w.Write(body.Bytes())
	}
}
