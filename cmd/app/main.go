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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed", "err", err)
		os.Exit(1)
	}
	log.Info("server stopped")
}
