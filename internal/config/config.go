package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration from environment variables.
type Config struct {
	// App
	Port      string
	StoreName string

	// OpenTelemetry (empty = no-op)
	OTELExporterEndpoint string
	OTELServiceName      string
}

const (
	defaultPort        = "8080"
	defaultStoreName   = "statestore"
	defaultOTELService = "example-go-app"
)

// Load loads .env from the working directory (if present) and returns Config.
// Environment variables take precedence over .env file values.
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:                 getEnv("APP_PORT", defaultPort),
		StoreName:            getEnv("STATESTORE_NAME", defaultStoreName),
		OTELExporterEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTELServiceName:      getEnv("OTEL_SERVICE_NAME", defaultOTELService),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
