# monitor-proxy

`monitor-proxy` is a lightweight HTTP proxy that forwards telemetry traffic to Grafana Cloud and can automatically pause forwarding when a configured billing alert is firing.

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
- `POST /webhook/grafana`
  - accepts webhook payload and updates blocked state immediately
- `POST /` (and other forwarded methods)
  - proxy endpoint for telemetry forwarding

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
