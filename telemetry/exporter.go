package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"golang.org/x/xerrors"
)

// TracerProvider creates a grpc otlp exporter and configures a trace provider.
// Caller is responsible for calling TracerProvider.Shutdown to ensure all data is flushed.
func TracerProvider(ctx context.Context, service string) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(service),
		),
	)

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(otlptracegrpc.WithInsecure()))
	if err != nil {
		return nil, xerrors.Errorf("creating otlp exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	return tracerProvider, nil
}
