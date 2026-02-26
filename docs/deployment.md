# Deployment Guide: Dapr Go Example App to AKS

This guide describes how to deploy the example Go application to Azure Kubernetes Service (AKS) with Dapr and Azure SQL Database as the state store.

## Prerequisites

- Azure CLI (`az`) logged in
- `kubectl` configured for your AKS cluster
- Dapr installed on the cluster (`dapr init -k`)
- Azure SQL Database created and accessible from the cluster

## Architecture

See [architecture.md](./architecture.md) for Mermaid diagrams.

### Application and Dapr State Store

The app uses a **Dapr State Store component** (Kubernetes `Component` resource) to persist state in Azure SQL. The Dapr sidecar reads the component config and connects to Azure SQL; the app never talks to the database directly.

### Observability (Logs, Traces, Metrics)

- **Logs**: The app uses **slog** and writes JSON logs to stdout. Container logs are collected by Promtail, Fluent Bit, Datadog Agent, or similar, and sent to Grafana Loki or Datadog Logs. No app configuration needed.
- **Traces and metrics**: The app emits **OpenTelemetry** traces and metrics for HTTP requests when `OTEL_EXPORTER_OTLP_ENDPOINT` is set. Data flows to an OTLP-compatible backend:

| Backend                                  | OTLP endpoint example       |
| ---------------------------------------- | --------------------------- |
| **Grafana** (Alloy + Tempo + Prometheus) | `http://alloy:4318`         |
| **Datadog Agent**                        | `http://datadog-agent:4318` |

## Step 1: Create Azure SQL Database

1. Create a resource group and SQL server (if not exists):

   ```bash
   az sql server create --name <server-name> --resource-group <rg> --location <location> \
     --admin-user <admin> --admin-password <password>
   ```

2. Create the database:

   ```bash
   az sql db create --resource-group <rg> --server <server-name> --name daprstate
   ```

3. Configure firewall to allow AKS egress IPs or use Private Endpoint for production.

## Step 2: Configure Dapr Azure SQL State Store Component

Create a Kubernetes secret with the connection string:

```bash
kubectl create secret generic azuresql-secret \
  --from-literal=connectionString='Server=tcp:<server>.database.windows.net,1433;Database=daprstate;User ID=<user>;Password=<password>;Encrypt=True;TrustServerCertificate=False;Connection Timeout=30;'
```

Apply the Dapr component. See [dapr-component.yaml](./dapr-component.yaml) for the full example.

For Azure AD authentication (recommended for production), use `useAzureAD: true` and configure the component with Managed Identity or service principal.

## Step 3: Build and Push Container Image

Build the application and push to a container registry (ACR, Docker Hub, etc.):

```bash
docker build -t <registry>/example-go-app:latest .
docker push <registry>/example-go-app:latest
```

## Step 4: Deploy to Kubernetes

Apply the deployment manifest. See [kubernetes-deployment.yaml](./kubernetes-deployment.yaml) for the example.

The deployment includes [Kubernetes health probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) on `/health`:

- **Startup probe**: Allows up to 60s for the app and Dapr sidecar to initialize before liveness/readiness run.
- **Readiness probe**: Removes the pod from Service endpoints when unhealthy (no traffic until ready).
- **Liveness probe**: Restarts the container when unrecoverable; uses a higher `failureThreshold` than readiness to avoid premature restarts.

Update the image reference and any environment variables, then:

```bash
kubectl apply -f docs/kubernetes-deployment.yaml
```

## Step 5: Verify Deployment

```bash
kubectl get pods -l app=example-go-app
kubectl logs -l app=example-go-app -c example-go-app -f
```

Test the API (from within the cluster or via port-forward):

```bash
# Save state
curl -X POST http://localhost:8080/api/v1/state/mykey -d '{"value":"hello"}'

# Get state
curl http://localhost:8080/api/v1/state/mykey

# Delete state
curl -X DELETE http://localhost:8080/api/v1/state/mykey

# Health check
curl http://localhost:8080/health
```

## Step 6: Configure Observability (Optional)

- **Logs**: The app writes JSON logs to stdout. Deploy a log collector (Promtail, Fluent Bit, Grafana Alloy, or Datadog Agent) to ship container logs to Grafana Loki or Datadog Logs. No app configuration needed.
- **Traces and metrics**: Set `OTEL_EXPORTER_OTLP_ENDPOINT` to your OTLP receiver:
  - **Grafana**: Deploy [Grafana Alloy](https://grafana.com/docs/alloy/latest/) or similar OTLP receiver; expose an OTLP HTTP endpoint (default port 4318). Point the app to it (e.g. `http://alloy.monitoring:4318`).
  - **Datadog**: Deploy the [Datadog Agent](https://docs.datadoghq.com/containers/kubernetes/apm/) with OTLP enabled; use the agent service (e.g. `http://datadog-agent.datadog:4318`).

Add the env vars to your Deployment (see [kubernetes-deployment.yaml](./kubernetes-deployment.yaml) for commented examples).

## Step 7: Deploy with ArgoCD (Optional)

To deploy via GitOps, create an ArgoCD Application. See [argocd-application.yaml](./argocd-application.yaml) for an example.

Ensure your Kubernetes manifests are in a Git repository, then apply the Application:

```bash
kubectl apply -f docs/argocd-application.yaml
```

## Summary

| Resource                          | Purpose                                                                                            |
| --------------------------------- | -------------------------------------------------------------------------------------------------- |
| **Dapr Component** (`statestore`) | Configures Azure SQL as the state store backend for Dapr                                           |
| **Dapr Sidecar**                  | Connects to Azure SQL via the component; app uses Dapr client for state                            |
| **Logs (slog)**                   | App writes JSON logs to stdout; collected by Promtail/Fluent Bit/Datadog Agent → Loki/Datadog Logs |
| **OpenTelemetry**                 | App emits traces/metrics to OTLP when `OTEL_EXPORTER_OTLP_ENDPOINT` is set                         |
| **Grafana / Datadog**             | OTLP backends for traces/metrics; Loki/Datadog Logs for logs                                       |

## Environment Variables

| Variable                      | Default          | Description                      |
| ----------------------------- | ---------------- | -------------------------------- |
| `APP_PORT`                    | `8080`           | HTTP server port                 |
| `STATESTORE_NAME`             | `statestore`     | Dapr state store component name  |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (none)           | OTLP endpoint for traces/metrics |
| `OTEL_SERVICE_NAME`           | `example-go-app` | Service name for telemetry       |

## Troubleshooting

- **Dapr sidecar not starting**: Check pod annotations (`dapr.io/enabled`, `dapr.io/app-id`, `dapr.io/app-port`).
- **State operations fail**: Verify the Dapr component is applied and the Azure SQL secret exists. Check Dapr sidecar logs: `kubectl logs <pod> -c daprd`.
- **No traces/metrics**: Ensure `OTEL_EXPORTER_OTLP_ENDPOINT` is set and the OTLP receiver (Grafana Alloy, Datadog Agent, etc.) is reachable from the pod.
- **Connection refused to Dapr**: Ensure the app listens on the port specified in `dapr.io/app-port`.
