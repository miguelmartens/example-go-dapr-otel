package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dapr/go-sdk/client"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/miguelmartens/example-go-dapr-otel/internal/config"
	"github.com/miguelmartens/example-go-dapr-otel/internal/server"
	"github.com/miguelmartens/example-go-dapr-otel/internal/telemetry"
)

// waitForDapr polls the Dapr outbound health endpoint until ready or ctx expires.
// Returns true if Dapr is ready. Skips waiting when DAPR_GRPC_PORT is unset (no sidecar injected).
func waitForDapr(ctx context.Context, log *slog.Logger) bool {
	if os.Getenv("DAPR_GRPC_PORT") == "" {
		return false // No sidecar (e.g. local dev); skip wait
	}
	port := "3500"
	if p := os.Getenv("DAPR_HTTP_PORT"); p != "" {
		port = p
	}
	url := "http://127.0.0.1:" + port + "/v1.0/healthz/outbound"
	httpClient := &http.Client{Timeout: 2 * time.Second}

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			select {
			case <-ctx.Done():
				return false
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			log.Info("Dapr sidecar ready")
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func main() {
	cfg := config.Load()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	shutdown := telemetry.Init(log, cfg.OTELExporterEndpoint, cfg.OTELServiceName)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	}()

	var stateClient server.StateClient
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	_ = waitForDapr(ctx, log)
	cancel()

	daprClient, err := client.NewClient()
	if err != nil {
		log.Info("Dapr unavailable, using in-memory store for local dev", "err", err)
		stateClient = server.NewMemStore()
	} else {
		defer daprClient.Close()
		stateClient = daprClient
	}

	srv := server.New(stateClient, cfg.StoreName, log)
	handler := otelhttp.NewHandler(srv.Handler(), "example-go-app")
	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", cfg.Port, "store", cfg.StoreName)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed", "err", err)
		os.Exit(1)
	}
	log.Info("server stopped")
}
