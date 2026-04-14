package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"metrics-proxy/internal/application/domain"
	"metrics-proxy/internal/application/services"
)

// Handler is the driving adapter that exposes HTTP endpoints.
type Handler struct {
	proxyService    *services.ProxyService
	alertPoller     *services.AlertPoller
	webhookSecret   string
	proxyAuthHeader string
	proxyAuthValue  string
}

func NewHandler(proxyService *services.ProxyService, alertPoller *services.AlertPoller, webhookSecret, proxyAuthHeader, proxyAuthValue string) *Handler {
	return &Handler{
		proxyService:    proxyService,
		alertPoller:     alertPoller,
		webhookSecret:   webhookSecret,
		proxyAuthHeader: proxyAuthHeader,
		proxyAuthValue:  proxyAuthValue,
	}
}

func (h *Handler) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	h.proxyService.HandleRequest(w, r)
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	blocked := h.proxyService.IsBlocked()
	status := "ok"
	if blocked {
		status = "blocked"
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"blocked": blocked,
	})
}

func (h *Handler) GrafanaWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.alertPoller == nil {
		http.Error(w, "alert polling is not configured", http.StatusServiceUnavailable)
		return
	}

	if !h.isWebhookAuthorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Grafana webhook payloads use "status": "firing"|"resolved" (string),
	// while the Alertmanager API uses "status": {"state": "..."} (object).
	var payload struct {
		Alerts []struct {
			Labels map[string]string `json:"labels"`
			Status string            `json:"status"`
		} `json:"alerts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	alerts := make([]domain.Alert, len(payload.Alerts))
	for i, a := range payload.Alerts {
		alerts[i] = domain.Alert{
			Labels: a.Labels,
			Status: struct {
				State string `json:"state"`
			}{State: a.Status},
		}
	}
	h.alertPoller.HandleWebhookAlerts(alerts)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"blocked": h.proxyService.IsBlocked(),
	})
}

func (h *Handler) isWebhookAuthorized(r *http.Request) bool {
	if h.webhookSecret == "" {
		return true
	}

	providedSecret := r.Header.Get("X-Webhook-Secret")
	if providedSecret == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			providedSecret = strings.TrimSpace(authHeader[len("Bearer "):])
		}
	}

	return subtle.ConstantTimeCompare([]byte(providedSecret), []byte(h.webhookSecret)) == 1
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.HealthHandler)
	mux.HandleFunc("/webhook/grafana", h.GrafanaWebhookHandler)
	mux.HandleFunc("/", CORSMiddleware(RequireProxyHeaderMiddleware(h.proxyAuthHeader, h.proxyAuthValue, h.ProxyHandler)))
}
