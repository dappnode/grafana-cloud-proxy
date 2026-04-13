package forwarder

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Adapter implements ports.MetricsForwarder by forwarding HTTP requests to a target URL.
type Adapter struct {
	targetURL string
	client    *http.Client
}

func NewAdapter(targetURL string) *Adapter {
	return &Adapter{
		targetURL: targetURL,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *Adapter) Forward(w http.ResponseWriter, r *http.Request) error {
	clientIP := getClientIP(r)
	log.Printf("Incoming request from IP: %s, Method: %s, Path: %s", clientIP, r.Method, r.URL.Path)

	proxyReq, err := http.NewRequest(r.Method, a.targetURL, r.Body)
	if err != nil {
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return fmt.Errorf("creating proxy request: %w", err)
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Remove hop-by-hop headers
	proxyReq.Header.Del("Connection")
	proxyReq.Header.Del("Proxy-Connection")
	proxyReq.Header.Del("Keep-Alive")
	proxyReq.Header.Del("Proxy-Authenticate")
	proxyReq.Header.Del("Proxy-Authorization")
	proxyReq.Header.Del("Te")
	proxyReq.Header.Del("Trailer")
	proxyReq.Header.Del("Transfer-Encoding")
	proxyReq.Header.Del("Upgrade")

	resp, err := a.client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusBadGateway)
		return fmt.Errorf("forwarding request: %w", err)
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		if isCORSHeader(key) {
			continue
		}
		w.Header()[key] = values
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v", err)
	}

	log.Printf("Request from %s completed with status: %d", clientIP, resp.StatusCode)
	return nil
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func isCORSHeader(headerKey string) bool {
	return strings.HasPrefix(strings.ToLower(headerKey), "access-control-")
}
