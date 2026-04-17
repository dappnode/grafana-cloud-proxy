# grafana-cloud-proxy

`grafana-cloud-proxy` is a lightweight HTTP proxy that forwards telemetry traffic to Grafana Cloud and can automatically pause forwarding when a configured billing alert is firing.

This proxy implementation was built following Grafana's frontend observability data-proxy guidance:

- https://grafana.com/docs/grafana-cloud/monitor-applications/frontend-observability/configure/data-proxy/

For broader product and platform concepts, use the main Grafana Cloud documentation as reference:

- https://grafana.com/docs/grafana-cloud/

It uses a hybrid control model:

- Grafana webhook updates for near real-time block/unblock state changes
- Periodic Grafana polling for reconciliation and startup initialization

## What This App Does

- Proxies incoming HTTP telemetry requests to `TARGET_URL`
- Applies CORS rules for DAppNode origins
- Exposes `GET /health`
- Exposes `POST /webhook/grafana` for Grafana alert notifications
- Blocks forwarding with HTTP `503` when the configured cost alert is firing

## Architecture

Hexagonal architecture with ports and adapters:

- `internal/application/domain`: domain entities
- `internal/application/ports`: app interfaces
- `internal/application/services`: core business logic (poller + proxy decision)
- `internal/adapters/grafana`: Grafana API client
- `internal/adapters/forwarder`: outbound HTTP forwarding adapter
- `internal/adapters/api`: inbound HTTP handlers and middleware
- `cmd/main.go`: dependency wiring and app bootstrap

## Configuration

The app is configured through environment variables.

Copy example values first:

```bash
cp .env.example .env
```

Required variables:

- `TARGET_URL`: upstream telemetry endpoint

Recommended variables:

- `GRAFANA_API_URL`: Grafana stack URL (example: `https://your-stack.grafana.net`)
- `GRAFANA_API_KEY`: Grafana API key used for polling active alerts
- `ALERT_NAME`: substring match used to identify the cost-control alert
- `POLL_INTERVAL`: reconciliation polling interval (default recommendation: `5m`)
- `GRAFANA_WEBHOOK_SECRET`: shared secret for webhook authentication
- `PROXY_AUTH_HEADER`: optional header name required on proxied requests
- `PROXY_AUTH_VALUE`: optional expected value for `PROXY_AUTH_HEADER`; if unset, only header presence is required
- `PORT`: HTTP server port (default `8080`)

## Run With Docker Compose (Production-like)

```bash
docker compose up --build
```

The main compose file loads runtime configuration from `.env`.

## Development Mode (Hot Reload)

### Option 1: Local (without Docker)

Install air once:

```bash
go install github.com/air-verse/air@latest
```

Run with hot reload:

```bash
air -c .air.toml
```

On every save in `cmd` or `internal`, the app rebuilds and restarts automatically.

### Option 2: Docker Compose Dev (hot reload in container)

```bash
docker compose -f docker-compose.dev.yml up --build
```

This mode mounts the source code into the container and runs air inside it.

## Webhook Setup In Grafana

Configure a webhook contact point that targets:

- `POST http://<host>:<port>/webhook/grafana`

Authentication:

- Set header `X-Webhook-Secret: <GRAFANA_WEBHOOK_SECRET>`
- Or use `Authorization: Bearer <GRAFANA_WEBHOOK_SECRET>`

Expected payload shape includes an `alerts` array compatible with Grafana Alertmanager webhook payloads.

## API Endpoints

- `GET /health`
  - returns app health and current blocked state
- `GET /metrics`
  - Prometheus metrics endpoint (see [Metrics](#metrics) below)
- `POST /webhook/grafana`
  - accepts webhook payload and updates blocked state immediately
- `POST /` (and other forwarded methods)
  - proxy endpoint for telemetry forwarding

## Metrics

The proxy exposes Prometheus metrics at `GET /metrics`. These are pushed to Grafana Cloud via a Grafana Alloy sidecar.

### Exposed Metrics

| Metric                           | Type      | Labels   | Description                                                         |
| -------------------------------- | --------- | -------- | ------------------------------------------------------------------- |
| `proxy_requests_total`           | Counter   | `status` | Total proxy requests by outcome                                     |
| `proxy_request_duration_seconds` | Histogram | —        | Latency of forwarded requests                                       |
| `proxy_blocked_by_alert`         | Gauge     | —        | Whether the proxy is currently blocked (`1`=blocked, `0`=unblocked) |

The `status` label on `proxy_requests_total` has four values:

- `forwarded` — request successfully proxied to the target
- `blocked` — request rejected with 503 because billing alert is firing
- `unauthorized` — request rejected with 401 due to missing or invalid auth header
- `error` — forwarding to the target failed (502)

### Grafana Alloy

A [Grafana Alloy](https://grafana.com/docs/alloy/) sidecar container scrapes the `/metrics` endpoint and pushes metrics to Grafana Cloud via Prometheus `remote_write`. The Alloy configuration is in `alloy/config.alloy`.

Alloy-specific environment variables:

- `ALLOY_REMOTE_WRITE_URL`: Grafana Cloud Prometheus remote write URL
- `ALLOY_REMOTE_WRITE_USER`: Grafana Cloud Prometheus username (numeric ID)
- `ALLOY_REMOTE_WRITE_PASS`: Grafana Cloud API token with `MetricsPublisher` role

Both `docker-compose.yml` and `docker-compose.dev.yml` include the Alloy sidecar. It scrapes the proxy every 15 seconds and forwards metrics to Grafana Cloud.

A pre-built dashboard JSON is available at `grafana/dashboard.json` and can be imported into Grafana Cloud via **Dashboards → New → Import**.

## Optional Proxy Header Gate

To apply a lightweight filter before forwarding telemetry requests, configure:

```bash
PROXY_AUTH_HEADER=X-Dappnode
PROXY_AUTH_VALUE=shared-secret
```

If `PROXY_AUTH_HEADER` is set, proxied requests must include that header. If `PROXY_AUTH_VALUE` is also set, the header must match exactly. This is only a simple gate and should not be treated as strong authentication.

Rejected proxy requests are logged with method, path, and client IP, and increment the `proxy_requests_total{status="unauthorized"}` metric.

## Testing

Run unit tests:

```bash
go test ./...
```

Run Grafana integration tests:

```bash
GRAFANA_API_URL=https://your-stack.grafana.net \
GRAFANA_API_KEY=glsa_your_key \
go test -tags=integration ./internal/adapters/grafana -v -count=1
```

## Security Notes

- Do not commit real API keys in source control
- Keep `.env` private (already gitignored)
- Rotate exposed credentials immediately if they were ever shared
- Use a strong webhook secret in production
