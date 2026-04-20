package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler is the driving adapter that exposes internal Prometheus metrics.
type MetricsHandler struct{}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

func (h *MetricsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())
}
