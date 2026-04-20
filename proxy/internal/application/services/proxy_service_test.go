package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metrics-proxy/internal/application/domain"
)

type blockingForwarder struct{}

func (f *blockingForwarder) Forward(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

func TestProxyService_Blocked(t *testing.T) {
	mock := &mockAlertProvider{alerts: []domain.Alert{}}
	p := NewAlertPoller(mock, "Global Spend", time.Minute)
	p.SetBlocked(true)

	svc := NewProxyService(p, &blockingForwarder{})

	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()

	svc.HandleRequest(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "metrics_paused" {
		t.Errorf("expected error=metrics_paused, got %q", body["error"])
	}
}
