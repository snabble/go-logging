package tracex

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TraceAndSpan adds the trace and span fields from context if available.
func TraceAndSpan(ctx context.Context, fields logrus.Fields) {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		fields["trace"] = spanContext.TraceID()
		fields["span"] = spanContext.SpanID()
	}
}

// BackgroundContextWithSpan creates a new context with the same span as ctx.
// This is useful for tracing functions that should not be interrupted if the caller cancels or times out.
func BackgroundContextWithSpan(ctx context.Context) context.Context {
	return trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx))
}

type Tracer = trace.Tracer

// GetTracer returns an unnamed tracer from the global trace provider.
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer("")
}

// GetNamedTracer returns a named tracer from the global trace provider.
func GetNamedTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
