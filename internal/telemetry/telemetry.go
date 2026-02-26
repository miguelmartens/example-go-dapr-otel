package telemetry

import (
	"context"
	"log/slog"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// Init initializes OpenTelemetry trace and metric providers.
// Uses cfg.OTELExporterEndpoint when set; otherwise no-op providers.
// Returns a shutdown function to flush and close exporters.
func Init(log *slog.Logger, endpoint, serviceName string) func(context.Context) error {
	if endpoint == "" {
		log.Info("OTEL_EXPORTER_OTLP_ENDPOINT not set, using no-op telemetry")
		return func(context.Context) error { return nil }
	}

	ctx := context.Background()

	tp, err := initTracer(ctx, endpoint, serviceName)
	if err != nil {
		log.Warn("failed to init tracer, using no-op", "err", err)
	} else {
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	}

	mp, err := initMeter(ctx, endpoint, serviceName)
	if err != nil {
		log.Warn("failed to init meter, using no-op", "err", err)
	} else {
		otel.SetMeterProvider(mp)
	}

	return func(ctx context.Context) error {
		var errs []error
		if tp != nil {
			if err := tp.Shutdown(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if mp != nil {
			if err := mp.Shutdown(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	}
}

func initTracer(ctx context.Context, endpoint, serviceName string) (*trace.TracerProvider, error) {
	host := parseEndpoint(endpoint)
	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(host),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(newResource(serviceName)),
	)
	return tp, nil
}

func initMeter(ctx context.Context, endpoint, serviceName string) (*metric.MeterProvider, error) {
	host := parseEndpoint(endpoint)
	exp, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(host),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exp)),
		metric.WithResource(newResource(serviceName)),
	)
	return mp, nil
}

func newResource(serviceName string) *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	return r
}

func parseEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "localhost:4318"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	if u.Host != "" {
		return u.Host
	}
	return endpoint
}
