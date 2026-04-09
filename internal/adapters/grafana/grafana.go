package grafana

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"metrics-proxy/internal/application/domain"
)

// Adapter implements ports.AlertProvider by calling the Grafana Alertmanager API.
type Adapter struct {
	grafanaURL string
	apiKey     string
	client     *http.Client
}

func NewAdapter(grafanaURL, apiKey string) *Adapter {
	return &Adapter{
		grafanaURL: strings.TrimRight(grafanaURL, "/"),
		apiKey:     apiKey,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *Adapter) FetchAlerts() ([]domain.Alert, error) {
	url := fmt.Sprintf("%s/api/alertmanager/grafana/api/v2/alerts", a.grafanaURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polling Grafana: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Grafana returned status %d: %s", resp.StatusCode, string(body))
	}

	var alerts []domain.Alert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return alerts, nil
}

func fetchRuleTitles(baseURL, apiKey string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/ruler/grafana/api/v1/rules", nil)
	if err != nil {
		return nil, fmt.Errorf("creating rules request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling rules API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rules API returned status %d", resp.StatusCode)
	}

	var payload map[string][]struct {
		Rules []struct {
			GrafanaAlert struct {
				Title string `json:"title"`
			} `json:"grafana_alert"`
		} `json:"rules"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding rules response: %w", err)
	}

	titles := make([]string, 0)
	for _, groups := range payload {
		for _, group := range groups {
			for _, rule := range group.Rules {
				if rule.GrafanaAlert.Title != "" {
					titles = append(titles, rule.GrafanaAlert.Title)
				}
			}
		}
	}

	return titles, nil
}
