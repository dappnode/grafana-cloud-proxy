package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts proxy requests by outcome: forwarded, blocked, unauthorized, error.
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_requests_total",
		Help: "Total number of proxy requests by outcome.",
	}, []string{"status"})

	// RequestDuration tracks latency of forwarded requests.
	RequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "proxy_request_duration_seconds",
		Help:    "Duration of forwarded proxy requests in seconds.",
		Buckets: prometheus.DefBuckets,
	})

	// AlertBlocked indicates whether the proxy is currently blocked by an alert (1=blocked, 0=unblocked).
	AlertBlocked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxy_blocked_by_alert",
		Help: "Whether the proxy is currently blocked by a billing alert (1=blocked, 0=unblocked).",
	})
)
