package services

import (
	"encoding/json"
	"log"
	"net/http"

	"metrics-proxy/internal/application/ports"
)

// ProxyService decides whether to forward a request or block it.
type ProxyService struct {
	Poller    *AlertPoller
	Forwarder ports.MetricsForwarder
}

func NewProxyService(poller *AlertPoller, forwarder ports.MetricsForwarder) *ProxyService {
	return &ProxyService{
		Poller:    poller,
		Forwarder: forwarder,
	}
}

func (s *ProxyService) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if s.Poller != nil && s.Poller.IsBlocked() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error":  "metrics_paused",
			"reason": "billing_threshold_reached",
		})
		return
	}

	if err := s.Forwarder.Forward(w, r); err != nil {
		log.Printf("Error forwarding request: %v", err)
	}
}

func (s *ProxyService) IsBlocked() bool {
	return s.Poller != nil && s.Poller.IsBlocked()
}
