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
	proxyService  *services.ProxyService
	alertPoller   *services.AlertPoller
	webhookSecret string
}

func NewHandler(proxyService *services.ProxyService, alertPoller *services.AlertPoller, webhookSecret string) *Handler {
	return &Handler{
		proxyService:  proxyService,
		alertPoller:   alertPoller,
		webhookSecret: webhookSecret,
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

	var payload struct {
		Alerts []domain.Alert `json:"alerts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	h.alertPoller.HandleWebhookAlerts(payload.Alerts)

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
	mux.HandleFunc("/", CORSMiddleware(h.ProxyHandler))
}
