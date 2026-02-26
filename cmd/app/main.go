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

	"github.com/miguelmartens/example-go-dapr-otel/internal/server"
	"github.com/miguelmartens/example-go-dapr-otel/internal/telemetry"
)

const defaultPort = "8080"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	shutdown := telemetry.Init(log)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	}()

	daprClient, err := client.NewClient()
	if err != nil {
		log.Error("failed to create Dapr client", "err", err)
		os.Exit(1)
	}
	defer daprClient.Close()

	storeName := getEnv("STATESTORE_NAME", "statestore")
	port := getEnv("APP_PORT", defaultPort)

	srv := server.New(daprClient, storeName, log)
	handler := otelhttp.NewHandler(srv.Handler(), "example-go-app")
	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", port, "store", storeName)
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
