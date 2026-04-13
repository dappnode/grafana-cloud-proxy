package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"metrics-proxy/internal/adapters/api"
	"metrics-proxy/internal/adapters/forwarder"
	"metrics-proxy/internal/adapters/grafana"
	"metrics-proxy/internal/application/services"
	"metrics-proxy/internal/config"
)

func main() {
	cfg := config.LoadConfig()

	// Create driven adapters
	fwd := forwarder.NewAdapter(cfg.TargetURL)

	// Create alert poller (optional — only if Grafana config is provided)
	var poller *services.AlertPoller
	if cfg.GrafanaURL != "" && cfg.GrafanaKey != "" && cfg.AlertName != "" {
		grafanaAdapter := grafana.NewAdapter(cfg.GrafanaURL, cfg.GrafanaKey)
		poller = services.NewAlertPoller(grafanaAdapter, cfg.AlertName, cfg.PollInterval)
		poller.Start()
	} else {
		log.Printf("WARNING: GRAFANA_API_URL, GRAFANA_API_KEY, or ALERT_NAME not set — billing alert polling disabled")
	}

	// Create application service
	proxySvc := services.NewProxyService(poller, fwd)

	// Create driving adapter (HTTP handler) and register routes
	handler := api.NewHandler(proxySvc, poller, cfg.WebhookSecret)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := ":" + cfg.Port
	log.Printf("Starting proxy server on port %s", cfg.Port)
	log.Printf("Proxying requests to: %s", cfg.TargetURL)
	if poller != nil {
		log.Printf("Alert reconciliation poll interval: %s", cfg.PollInterval)
	}
	log.Printf("Allowed CORS origins: my.dappnode, dappmanager.dappnode, dappmanager.dappnode.private, my.dappnode.private")

	server := &http.Server{Addr: addr, Handler: mux}

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	if poller != nil {
		poller.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
