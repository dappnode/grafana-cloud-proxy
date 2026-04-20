package ports

import "metrics-proxy/internal/application/domain"

// AlertProvider is the port for fetching alerts from an external alerting system.
type AlertProvider interface {
	FetchAlerts() ([]domain.Alert, error)
}
