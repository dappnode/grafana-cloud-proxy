package ports

import "net/http"

// MetricsForwarder is the port for forwarding HTTP requests to the metrics target.
type MetricsForwarder interface {
	Forward(w http.ResponseWriter, r *http.Request) error
}
