//go:build integration

package grafana

import (
	"os"
	"strings"
	"testing"
)

func requireIntegrationEnv(t *testing.T) (string, string) {
	t.Helper()

	url := strings.TrimSpace(os.Getenv("GRAFANA_API_URL"))
	key := strings.TrimSpace(os.Getenv("GRAFANA_API_KEY"))

	if url == "" || key == "" {
		t.Skip("integration env vars not set: GRAFANA_API_URL and GRAFANA_API_KEY")
	}

	return url, key
}

func TestFetchAlerts_Integration_RealGrafana(t *testing.T) {
	url, key := requireIntegrationEnv(t)

	adapter := NewAdapter(url, key)
	alerts, err := adapter.FetchAlerts()
	if err != nil {
		t.Fatalf("FetchAlerts real query failed: %v", err)
	}

	t.Logf("fetched %d alerts from Grafana", len(alerts))
	if len(alerts) == 0 {
		t.Log("no active alerts returned by Grafana")
	}

	for i, alert := range alerts {
		if alert.Labels == nil {
			t.Fatalf("alert[%d] has nil labels", i)
		} else {
			t.Logf("alert[%d] labels: %v", i, alert.Labels)
		}
	}
}

func TestRules_Integration_ConfiguredRules(t *testing.T) {
	url, key := requireIntegrationEnv(t)

	titles, err := fetchRuleTitles(url, key)
	if err != nil {
		t.Fatalf("fetching configured rules failed: %v", err)
	}

	t.Logf("configured rules in Grafana: %d", len(titles))
	for i, title := range titles {
		t.Logf("rule[%d]: %s", i, title)
	}
}

func TestFetchAlerts_Integration_InvalidKey(t *testing.T) {
	url, _ := requireIntegrationEnv(t)

	adapter := NewAdapter(url, "invalid-token-for-integration-test")
	_, err := adapter.FetchAlerts()
	if err == nil {
		t.Fatal("expected error with invalid Grafana API key")
	}
}
