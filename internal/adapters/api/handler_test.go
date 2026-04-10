package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metrics-proxy/internal/application/domain"
	"metrics-proxy/internal/application/services"
)

type mockAlertProvider struct {
	alerts []domain.Alert
	err    error
}

func (m *mockAlertProvider) FetchAlerts() ([]domain.Alert, error) {
	return m.alerts, m.err
}

type noopForwarder struct{}

func (f *noopForwarder) Forward(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

func newHandler(blocked bool) *Handler {
	mock := &mockAlertProvider{alerts: []domain.Alert{}}
	poller := services.NewAlertPoller(mock, "Global Spend", time.Minute)
	poller.SetBlocked(blocked)
	svc := services.NewProxyService(poller, &noopForwarder{})
	return NewHandler(svc, poller, "")
}

func TestHealthHandler_Unblocked(t *testing.T) {
	h := newHandler(false)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
	if body["blocked"] != false {
		t.Errorf("expected blocked=false")
	}
}

func TestHealthHandler_Blocked(t *testing.T) {
	h := newHandler(true)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.HealthHandler(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "blocked" {
		t.Errorf("expected status=blocked, got %q", body["status"])
	}
	if body["blocked"] != true {
		t.Errorf("expected blocked=true")
	}
}

func TestGrafanaWebhookHandler_UpdatesBlockedState(t *testing.T) {
	h := newHandler(false)

	body := []byte(`{"alerts":[{"labels":{"alertname":"Global Spend: 85% of $1"},"status":"firing"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.GrafanaWebhookHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !h.proxyService.IsBlocked() {
		t.Fatal("expected proxy to be blocked after firing webhook")
	}
}

func TestGrafanaWebhookHandler_RequiresSecret(t *testing.T) {
	mock := &mockAlertProvider{alerts: []domain.Alert{}}
	poller := services.NewAlertPoller(mock, "Global Spend", time.Minute)
	svc := services.NewProxyService(poller, &noopForwarder{})
	h := NewHandler(svc, poller, "topsecret")

	body := []byte(`{"alerts":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.GrafanaWebhookHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
