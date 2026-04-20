package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	GrafanaURL      string
	GrafanaKey      string
	AlertName       string
	PollInterval    time.Duration
	WebhookSecret   string
	ProxyAuthHeader string
	ProxyAuthValue  string
	Port            string
	MetricsPort     string
	TargetURL       string
}

func LoadConfig() Config {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		log.Fatal("TARGET_URL environment variable is required")
	}

	port := getEnv("PORT", "8080")
	metricsPort := getEnv("METRICS_PORT", "9090")

	var pollInterval time.Duration
	intervalStr := getEnv("POLL_INTERVAL", "5m")
	var err error
	pollInterval, err = time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("Invalid POLL_INTERVAL: %v", err)
	}

	return Config{
		GrafanaURL:      os.Getenv("GRAFANA_API_URL"),
		GrafanaKey:      os.Getenv("GRAFANA_API_KEY"),
		AlertName:       "proxy-test",//os.Getenv("ALERT_NAME"),
		PollInterval:    pollInterval,
		WebhookSecret:   os.Getenv("GRAFANA_WEBHOOK_SECRET"),
		ProxyAuthHeader: os.Getenv("PROXY_AUTH_HEADER"),
		ProxyAuthValue:  os.Getenv("PROXY_AUTH_VALUE"),
		Port:            port,
		MetricsPort:     metricsPort,
		TargetURL:       targetURL,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
