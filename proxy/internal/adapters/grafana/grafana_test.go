package grafana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics-proxy/internal/application/domain"
)

func TestFetchAlerts(t *testing.T) {
	wantAlerts := []domain.Alert{
		{Labels: map[string]string{"alertname": "Global Spend: 50% of $1", "product": "Global Spend"}},
		{Labels: map[string]string{"alertname": "Global Spend: 85% of $1", "product": "Global Spend"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/alertmanager/grafana/api/v2/alerts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(wantAlerts)
	}))
	defer server.Close()

	adapter := NewAdapter(server.URL, "test-key")
	got, err := adapter.FetchAlerts()
	if err != nil {
		t.Fatalf("FetchAlerts returned error: %v", err)
	}

	if len(got) != len(wantAlerts) {
		t.Fatalf("expected %d alerts, got %d", len(wantAlerts), len(got))
	}
	for i, alert := range got {
		if alert.Labels["alertname"] != wantAlerts[i].Labels["alertname"] {
			t.Errorf("alert[%d] name = %q, want %q", i, alert.Labels["alertname"], wantAlerts[i].Labels["alertname"])
		}
	}
}

func TestFetchAlerts_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid token"))
	}))
	defer server.Close()

	adapter := NewAdapter(server.URL, "bad-key")
	_, err := adapter.FetchAlerts()
	if err == nil {
		t.Fatal("expected error from FetchAlerts with 401 response")
	}
}
