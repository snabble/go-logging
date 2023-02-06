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

type Tracer = trace.Tracer

// GetTracer returns the default otel tracer. You probably want to use NewTracerProvider instead.
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer("")
}
