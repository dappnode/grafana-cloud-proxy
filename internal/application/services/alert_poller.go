package services

import (
	"log"
	"strings"
	"sync/atomic"
	"time"

	"metrics-proxy/internal/application/domain"
	"metrics-proxy/internal/application/ports"
)

// AlertPoller polls an AlertProvider to check if a billing alert is firing.
type AlertPoller struct {
	provider     ports.AlertProvider
	alertName    string
	pollInterval time.Duration
	blocked      atomic.Bool
	stop         chan struct{}
}

func NewAlertPoller(provider ports.AlertProvider, alertName string, pollInterval time.Duration) *AlertPoller {
	return &AlertPoller{
		provider:     provider,
		alertName:    alertName,
		pollInterval: pollInterval,
		stop:         make(chan struct{}),
	}
}

func (p *AlertPoller) IsBlocked() bool {
	return p.blocked.Load()
}

func (p *AlertPoller) SetBlocked(v bool) {
	p.blocked.Store(v)
}

func (p *AlertPoller) CheckAlertState() {
	alerts, err := p.provider.FetchAlerts()
	if err != nil {
		log.Printf("[AlertPoller] %v", err)
		return
	}

	p.setBlockedStateFromAlerts(alerts, "poll")
}

// HandleWebhookAlerts updates state immediately from Grafana webhook payload.
func (p *AlertPoller) HandleWebhookAlerts(alerts []domain.Alert) {
	p.setBlockedStateFromAlerts(alerts, "webhook")
}

func (p *AlertPoller) setBlockedStateFromAlerts(alerts []domain.Alert, source string) {
	firing := false
	for _, alert := range alerts {
		if p.alertName == "" || !strings.Contains(alert.Labels["alertname"], p.alertName) {
			continue
		}

		if strings.EqualFold(alert.Status.State, "resolved") {
			continue
		}

		firing = true
		break
	}

	wasPreviouslyBlocked := p.blocked.Load()
	p.blocked.Store(firing)

	if firing && !wasPreviouslyBlocked {
		log.Printf("[AlertPoller] (%s) Alert '%s' is FIRING — blocking proxy", source, p.alertName)
	} else if !firing && wasPreviouslyBlocked {
		log.Printf("[AlertPoller] (%s) Alert '%s' is RESOLVED — unblocking proxy", source, p.alertName)
	}
}

// Start performs an initial check then polls on a ticker in a background goroutine.
func (p *AlertPoller) Start() {
	log.Printf("[AlertPoller] Checking alert state on startup...")
	p.CheckAlertState()
	if p.IsBlocked() {
		log.Printf("[AlertPoller] Proxy starts BLOCKED (alert '%s' is firing)", p.alertName)
	} else {
		log.Printf("[AlertPoller] Proxy starts UNBLOCKED (alert '%s' is not firing)", p.alertName)
	}

	go func() {
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.CheckAlertState()
			case <-p.stop:
				return
			}
		}
	}()
}

// Stop signals the polling goroutine to exit.
func (p *AlertPoller) Stop() {
	close(p.stop)
}
