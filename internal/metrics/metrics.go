// Package metrics defines the Prometheus metrics the bot exports.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics records what the bot does. A nil *Metrics records nothing.
type Metrics struct {
	registry *prometheus.Registry
	updates  *prometheus.CounterVec
	commands *prometheus.CounterVec
	sends    *prometheus.CounterVec
	latency  prometheus.Histogram
}

// New returns Metrics backed by its own registry.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		updates: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "updates_total",
			Help: "Telegram updates handled, by outcome.",
		}, []string{"outcome"}),
		commands: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "commands_total",
			Help: "Commands executed, by name.",
		}, []string{"name"}),
		sends: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "telegram_send_total",
			Help: "Calls that send to Telegram, by status.",
		}, []string{"status"}),
		latency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "update_duration_seconds",
			Help:    "Time spent handling an update, including replies.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	m.registry.MustRegister(
		m.updates, m.commands, m.sends, m.latency,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// Handler serves the metrics in the Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// GaugeFunc registers a gauge whose value is read at scrape time.
func (m *Metrics) GaugeFunc(name, help string, value func() float64) {
	m.registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: name, Help: help}, value))
}

// ObserveUpdate records one handled update. command may be empty.
func (m *Metrics) ObserveUpdate(outcome, command string, elapsed time.Duration) {
	if m == nil {
		return
	}
	m.updates.WithLabelValues(outcome).Inc()
	if command != "" {
		m.commands.WithLabelValues(command).Inc()
	}
	m.latency.Observe(elapsed.Seconds())
}

// ObserveSend records the result of one send to Telegram.
func (m *Metrics) ObserveSend(err error) {
	if m == nil {
		return
	}
	status := "ok"
	if err != nil {
		status = "error"
	}
	m.sends.WithLabelValues(status).Inc()
}
