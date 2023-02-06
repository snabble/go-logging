package tracex

import (
	"context"

	"github.com/sirupsen/logrus"
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
