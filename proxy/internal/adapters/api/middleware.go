package api

import (
	"crypto/subtle"
	"log"
	"net"
	"net/http"
	"strings"

	"metrics-proxy/internal/metrics"
)

var allowedOrigins = map[string]bool{
	"http://my.dappnode":                   true,
	"https://my.dappnode":                  true,
	"http://dappmanager.dappnode":          true,
	"https://dappmanager.dappnode":         true,
	"http://dappmanager.dappnode.private":  true,
	"https://dappmanager.dappnode.private": true,
	"http://my.dappnode.private":           true,
	"https://my.dappnode.private":          true,
}

func isOriginAllowed(origin string) bool {
	return allowedOrigins[origin]
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, x-faro-session-id, X-Dappnode")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			if isOriginAllowed(origin) {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		next(w, r)
	}
}

func clientIPFromRequest(r *http.Request) string {
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

func RequireProxyHeaderMiddleware(headerName, expectedValue string, next http.HandlerFunc) http.HandlerFunc {
	if strings.TrimSpace(headerName) == "" {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		providedValue := strings.TrimSpace(r.Header.Get(headerName))
		if providedValue == "" {
			metrics.RequestsTotal.WithLabelValues("unauthorized").Inc()
			log.Printf("Blocked proxy request: missing required header %q method=%s path=%s ip=%s", headerName, r.Method, r.URL.Path, clientIPFromRequest(r))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if expectedValue != "" && subtle.ConstantTimeCompare([]byte(providedValue), []byte(expectedValue)) != 1 {
			metrics.RequestsTotal.WithLabelValues("unauthorized").Inc()
			log.Printf("Blocked proxy request: invalid header %q value method=%s path=%s ip=%s", headerName, r.Method, r.URL.Path, clientIPFromRequest(r))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
