# Example Go App with Dapr and OpenTelemetry

A minimal Go application demonstrating Dapr SDK usage with Azure SQL Database as a state store, and OpenTelemetry for traces and metrics. When deployed to AKS with Dapr, the app uses the Dapr sidecar to persist state in Azure SQL and emits OTLP telemetry when configured.

## Requirements

- Go 1.26+
- Dapr CLI (optional; only needed for local dev with Dapr sidecar)

## Project Layout

```
.
├── cmd/app/           # Application entry point
├── internal/config/   # Environment and configuration
├── internal/server/   # HTTP handlers and Dapr state logic
├── internal/telemetry # OpenTelemetry trace/metric init
├── components/        # Dapr components (for local dev with Dapr)
├── docs/              # Deployment examples (Dapr, Kubernetes, ArgoCD)
├── .env.example       # Env template; copy to .env for local config
├── Makefile
└── go.mod
```

## Quick Start

### Local (without Dapr)

The app runs locally without Dapr by default. When the Dapr sidecar is unavailable, it automatically falls back to an in-memory state store. No Docker or Dapr setup required.

For optional local config (e.g. OpenTelemetry to a local collector), copy `.env.example` to `.env` and adjust:

```bash
cp .env.example .env
# Edit .env to set OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 etc.
make dev
```

Then test the API:

```bash
curl -X POST http://localhost:8080/api/v1/state/foo -d 'hello'
curl http://localhost:8080/api/v1/state/foo
curl -X DELETE http://localhost:8080/api/v1/state/foo
curl http://localhost:8080/health
```

### Local (with Dapr)

For local development with the Dapr sidecar (e.g. to test against Redis or another state store):

1. Start Dapr with a state store (e.g., Redis):

   ```bash
   dapr run --app-id example-go-app --app-port 8080 -- go run ./cmd/app
   ```

2. Or run the built binary:

   ```bash
   make run
   ```

3. Test the API:

   ```bash
   curl -X POST http://localhost:8080/api/v1/state/foo -d 'hello'
   curl http://localhost:8080/api/v1/state/foo
   curl -X DELETE http://localhost:8080/api/v1/state/foo
   curl http://localhost:8080/health
   ```

### Build

```bash
make build
./bin/app
```

## API

| Method | Path                  | Description                                   |
| ------ | --------------------- | --------------------------------------------- |
| GET    | `/livez`              | Liveness probe (should process be restarted?) |
| GET    | `/readyz`             | Readiness probe (ready to accept traffic?)    |
| GET    | `/health`             | Alias for `/readyz` (backwards compatibility) |
| GET    | `/api/v1/state/{key}` | Retrieve state value                          |
| POST   | `/api/v1/state/{key}` | Save state (body = value)                     |
| DELETE | `/api/v1/state/{key}` | Delete state                                  |

## Configuration

| Environment                   | Default          | Description                                                     |
| ----------------------------- | ---------------- | --------------------------------------------------------------- |
| `APP_PORT`                    | `8080`           | HTTP server port                                                |
| `STATESTORE_NAME`             | `statestore`     | Dapr state store component name                                 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (none)           | OTLP endpoint for traces/metrics (e.g. `http://localhost:4318`) |
| `OTEL_SERVICE_NAME`           | `example-go-app` | Service name for telemetry                                      |

**Local dev**: The app loads `.env` from the working directory if present. Copy `.env.example` to `.env` and customize. `.env` is gitignored.

## Observability

- **Logs**: The app uses slog and writes JSON logs to stdout. Container logs are collected by Promtail, Fluent Bit, Datadog Agent, or similar, and sent to Grafana Loki or Datadog Logs. No app configuration needed.
- **Traces**: HTTP request spans with method, path, status code (when `OTEL_EXPORTER_OTLP_ENDPOINT` is set)
- **Metrics**: Request count and duration from otelhttp middleware (when `OTEL_EXPORTER_OTLP_ENDPOINT` is set)

Compatible with Grafana (Loki + OTLP/Tempo), Datadog (Logs + OTLP), and other OTLP backends. When the OTLP endpoint is not set, traces and metrics are disabled (no-op).

## Deployment to AKS

See [docs/deployment.md](docs/deployment.md) for step-by-step instructions to deploy to AKS with Azure SQL. Architecture diagrams: [docs/architecture.md](docs/architecture.md). Example manifests:

- [docs/dapr-component.yaml](docs/dapr-component.yaml) – Azure SQL state store component
- [docs/kubernetes-deployment.yaml](docs/kubernetes-deployment.yaml) – Deployment and Service
- [docs/argocd-application.yaml](docs/argocd-application.yaml) – ArgoCD Application

## Development

```bash
make dev      # Clean, build, and run (local dev without Dapr)
make build    # Build binary
make run      # Build and run
make test     # Run tests
make lint     # Run go vet and golangci-lint
make fmt      # Format code
make tidy     # Tidy go.mod
make clean    # Remove build artifacts
```

**Local dev without Dapr**: `make dev` runs the app with an in-memory state store when Dapr is unavailable. No Docker or Dapr required.

## Tooling

- **golangci-lint**: Linting (see `.golangci.yml`)
- **Prettier**: JSON, YAML, Markdown (see `.prettierrc`)
- **Renovate**: Dependency updates (see `renovate.json`)
- **.gitattributes**: LF line endings
