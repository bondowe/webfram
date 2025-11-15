// Package telemetry provides Prometheus-based metrics collection and HTTP handler support for the framework.
package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//nolint:gochecknoglobals // Package-level registry and metrics required by Prometheus
var (
	registry *prometheus.Registry

	// RequestsTotal counts the total number of HTTP requests received.
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests received",
		},
		[]string{"method", "path", "status"},
	)

	// RequestDurationSeconds measures the duration of HTTP requests.
	RequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// ActiveConnections tracks the current number of active connections.
	ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Current number of active connections",
		},
	)
)

// ConfigureTelemetry initializes the telemetry registry and registers the provided collectors.
// If useDefaultRegistry is true, uses the default Prometheus registry; otherwise creates a new one.
func ConfigureTelemetry(useDefaultRegistry bool, collectors ...prometheus.Collector) {
	if useDefaultRegistry {
		var ok bool
		registry, ok = prometheus.DefaultRegisterer.(*prometheus.Registry)
		if !ok {
			registry = prometheus.NewRegistry()
		}
	} else {
		registry = prometheus.NewRegistry()
	}

	if len(collectors) > 0 {
		registry.MustRegister(collectors...)
	} else {
		registry.MustRegister(
			RequestsTotal,
			RequestDurationSeconds,
			ActiveConnections,
		)
	}
}

// GetHTTPHandler returns an HTTP handler for the prometheus metrics endpoint.
func GetHTTPHandler(opts promhttp.HandlerOpts) http.Handler {
	return promhttp.HandlerFor(
		registry,
		opts,
	)
}
