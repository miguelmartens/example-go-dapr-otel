# Architecture

## Application and Dapr State Store

The app uses a **Dapr State Store component** (Kubernetes `Component` resource) to persist state in Azure SQL. The Dapr sidecar reads the component config and connects to Azure SQL; the app never talks to the database directly.

```mermaid
flowchart TB
    subgraph Component [Dapr Component - statestore]
        Comp["kind: Component<br/>spec.type: state.sqlserver<br/>metadata: connectionString, etc."]
    end

    subgraph Pod [Pod: example-go-app]
        App["Go HTTP Server<br/>:8080"]
        Sidecar["Dapr Sidecar (daprd)<br/>:50001"]
    end

    DB[(Azure SQL Database<br/>state store backend)]

    Comp -->|configures| Sidecar
    App -->|gRPC| Sidecar
    Sidecar -->|state.sqlserver| DB
```

## Observability (Logs, Traces, Metrics)

### Logs

The app uses **slog** and writes JSON logs to stdout. Container logs are collected by a log collector (Promtail, Fluent Bit, Datadog Agent, etc.) and sent to Grafana Loki or Datadog Logs.

```mermaid
flowchart LR
    subgraph AppPod [Pod: example-go-app]
        App["Go HTTP Server"]
        Slog["slog (JSON)"]
    end

    subgraph Collectors [Log Collectors]
        Promtail["Promtail / Fluent Bit"]
        DDAgent["Datadog Agent"]
    end

    subgraph LogBackends [Log Backends]
        Loki["Grafana Loki"]
        DDLogs["Datadog Logs"]
    end

    App --> Slog
    Slog -->|stdout| Promtail
    Slog -->|stdout| DDAgent
    Promtail --> Loki
    DDAgent --> DDLogs
```

### Traces and Metrics (OpenTelemetry)

The app emits **OpenTelemetry** traces and metrics for HTTP requests when `OTEL_EXPORTER_OTLP_ENDPOINT` is set. Data flows to an OTLP-compatible backend (Grafana or Datadog).

```mermaid
flowchart LR
    subgraph AppPod [Pod: example-go-app]
        HTTP["Go HTTP Server"]
        OTel["otelhttp middleware"]
    end

    subgraph OTLPBackends [OTLP Backends]
        Grafana["Grafana<br/>Alloy + Tempo + Prometheus"]
        Datadog["Datadog Agent"]
    end

    HTTP --> OTel
    OTel -->|OTLP traces + metrics| Grafana
    OTel -->|OTLP traces + metrics| Datadog
```

## End-to-End Flow

```mermaid
flowchart TB
    subgraph AKS [AKS Cluster]
        subgraph AppPod [Pod: example-go-app]
            App["Go HTTP Server :8080"]
            Sidecar["Dapr Sidecar :50001"]
        end

        Component["Dapr Component<br/>statestore"]
    end

    AzureSQL[(Azure SQL Database)]
    OTLP["OTLP Collector<br/>Grafana / Datadog"]
    Logs["Log Collector<br/>Loki / Datadog"]

    Client["HTTP Client"] --> App
    App -->|gRPC| Sidecar
    Component -->|configures| Sidecar
    Sidecar -->|state.sqlserver| AzureSQL
    App -->|traces + metrics| OTLP
    App -->|logs stdout| Logs
```
