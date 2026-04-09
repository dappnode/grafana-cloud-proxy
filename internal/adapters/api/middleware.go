package api

import (
	"net/http"
	"strings"
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

func isCORSHeader(headerKey string) bool {
	return strings.HasPrefix(strings.ToLower(headerKey), "access-control-")
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, x-faro-session-id")
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
