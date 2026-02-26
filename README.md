# Example Go App with Dapr and OpenTelemetry

A minimal Go application demonstrating Dapr SDK usage with Azure SQL Database as a state store, and OpenTelemetry for traces and metrics. When deployed to AKS with Dapr, the app uses the Dapr sidecar to persist state in Azure SQL and emits OTLP telemetry when configured.

## Requirements

- Go 1.26+
- Dapr CLI (for local development)

## Project Layout

```
.
├── cmd/app/           # Application entry point
├── internal/server/   # HTTP handlers and Dapr state logic
├── internal/telemetry # OpenTelemetry trace/metric init
├── docs/              # Deployment examples (Dapr, Kubernetes, ArgoCD)
├── Makefile
└── go.mod
```

## Quick Start

### Local (with Dapr)

1. Start Dapr with a state store (e.g., Redis):

   ```bash
   dapr run --app-id example-go-app --app-port 8080 -- go run ./cmd/app
   ```

2. Run the app:

   ```bash
   make run
   ```

3. Test the API:

   ```bash
   curl -X POST http://localhost:8080/state/foo -d 'hello'
   curl http://localhost:8080/state/foo
   curl -X DELETE http://localhost:8080/state/foo
   curl http://localhost:8080/health
   ```

### Build

```bash
make build
./bin/app
```

## API

| Method | Path           | Description                        |
| ------ | -------------- | ---------------------------------- |
| GET    | `/health`      | Health check for Kubernetes probes |
| GET    | `/state/{key}` | Retrieve state value               |
| POST   | `/state/{key}` | Save state (body = value)          |
| DELETE | `/state/{key}` | Delete state                       |

## Configuration

| Environment                   | Default          | Description                                                     |
| ----------------------------- | ---------------- | --------------------------------------------------------------- |
| `APP_PORT`                    | `8080`           | HTTP server port                                                |
| `STATESTORE_NAME`             | `statestore`     | Dapr state store component name                                 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (none)           | OTLP endpoint for traces/metrics (e.g. `http://localhost:4318`) |
| `OTEL_SERVICE_NAME`           | `example-go-app` | Service name for telemetry                                      |

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
make build    # Build binary
make run      # Build and run
make test     # Run tests
make lint     # Run go vet and golangci-lint
make fmt      # Format code
make tidy     # Tidy go.mod
make clean    # Remove build artifacts
```

## Tooling

- **golangci-lint**: Linting (see `.golangci.yml`)
- **Prettier**: JSON, YAML, Markdown (see `.prettierrc`)
- **Renovate**: Dependency updates (see `renovate.json`)
- **.gitattributes**: LF line endings
