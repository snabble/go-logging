package tracex

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace/noop"
)

type TraceProvider struct {
	tp *sdktrace.TracerProvider
}

func NewGlobalNoopTraceProvider(serviceName, serviceSemanticVersion string) *TraceProvider {
	provider := NewTraceProvider(
		newResource(serviceName, serviceSemanticVersion, environment()),
		tracetest.NewNoopExporter(),
	)
	otel.SetTracerProvider(provider.tp)
	otel.SetTextMapPropagator(newTextMapPropagator())
	return provider
}

func NewTraceProvider(appResource *resource.Resource, batcher sdktrace.SpanExporter) *TraceProvider {
	return &TraceProvider{
		tp: sdktrace.NewTracerProvider(sdktrace.WithResource(appResource), sdktrace.WithBatcher(batcher)),
	}
}

func (p *TraceProvider) Shutdown(ctx context.Context) error {
	err := p.tp.Shutdown(ctx)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
	otel.SetTracerProvider(noop.NewTracerProvider())
	return err
}

func newTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
}

func newResource(serviceName, serviceSemVersion, environment string) *resource.Resource {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceSemVersion),
			attribute.String("environment", environment),
		),
	)
	if err != nil {
		panic(err)
	}
	return r
}

func environment() string {
	if env, ok := os.LookupEnv("ENV_NAME"); ok {
		return env
	}
	return "unknown"
}
