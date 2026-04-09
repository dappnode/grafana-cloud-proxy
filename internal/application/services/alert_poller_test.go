package services

import (
	"fmt"
	"testing"
	"time"

	"metrics-proxy/internal/application/domain"
)

// mockAlertProvider is a test double for ports.AlertProvider.
type mockAlertProvider struct {
	alerts []domain.Alert
	err    error
}

func (m *mockAlertProvider) FetchAlerts() ([]domain.Alert, error) {
	return m.alerts, m.err
}

func newTestAlerts(alertName string) []domain.Alert {
	return []domain.Alert{
		{Labels: map[string]string{"alertname": alertName}},
	}
}

func TestAlertPoller_Firing(t *testing.T) {
	mock := &mockAlertProvider{alerts: newTestAlerts("Global Spend: 50% of $1")}
	p := NewAlertPoller(mock, "Global Spend", time.Minute)
	p.CheckAlertState()

	if !p.IsBlocked() {
		t.Error("expected poller to be blocked when alert is firing")
	}
}

func TestAlertPoller_Resolved(t *testing.T) {
	mock := &mockAlertProvider{alerts: []domain.Alert{}}
	p := NewAlertPoller(mock, "Global Spend", time.Minute)
	p.SetBlocked(true)
	p.CheckAlertState()

	if p.IsBlocked() {
		t.Error("expected poller to be unblocked when no matching alert")
	}
}

func TestAlertPoller_DifferentAlertName(t *testing.T) {
	mock := &mockAlertProvider{alerts: newTestAlerts("some_other_alert")}
	p := NewAlertPoller(mock, "Global Spend", time.Minute)
	p.CheckAlertState()

	if p.IsBlocked() {
		t.Error("expected poller to NOT be blocked when alert name doesn't match")
	}
}

func TestAlertPoller_APIError(t *testing.T) {
	mock := &mockAlertProvider{err: fmt.Errorf("connection refused")}
	p := NewAlertPoller(mock, "Global Spend", time.Minute)
	p.SetBlocked(true)
	p.CheckAlertState()

	if !p.IsBlocked() {
		t.Error("expected blocked state to remain unchanged on API error")
	}
}

func TestAlertPoller_WebhookFiring(t *testing.T) {
	mock := &mockAlertProvider{alerts: []domain.Alert{}}
	p := NewAlertPoller(mock, "Global Spend", time.Minute)
	p.HandleWebhookAlerts(newTestAlerts("Global Spend: 85% of $1"))

	if !p.IsBlocked() {
		t.Error("expected poller to be blocked when webhook reports firing")
	}
}

func TestAlertPoller_WebhookResolved(t *testing.T) {
	mock := &mockAlertProvider{alerts: []domain.Alert{}}
	p := NewAlertPoller(mock, "Global Spend", time.Minute)
	p.SetBlocked(true)

	alerts := newTestAlerts("Global Spend: 85% of $1")
	alerts[0].Status.State = "resolved"
	p.HandleWebhookAlerts(alerts)

	if p.IsBlocked() {
		t.Error("expected poller to be unblocked when webhook reports resolved")
	}
}
