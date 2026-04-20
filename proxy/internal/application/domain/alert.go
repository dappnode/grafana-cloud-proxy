package domain

// Alert represents a firing alert from Grafana Alertmanager.
type Alert struct {
	Labels map[string]string `json:"labels"`
	Status struct {
		State string `json:"state"`
	} `json:"status"`
}
