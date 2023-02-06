package tracex

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
)

type traceHeaderPropagation struct {
	propagation propagation.TraceContext
}

func (t *traceHeaderPropagation) Inject(ctx context.Context, carrier http.Header) {
	t.propagation.Inject(ctx, propagation.HeaderCarrier(carrier))
}

func (t *traceHeaderPropagation) Extract(ctx context.Context, carrier http.Header) context.Context {
	return t.propagation.Extract(ctx, propagation.HeaderCarrier(carrier))
}

func (t *traceHeaderPropagation) Fields() []string {
	return t.Fields()
}

type TraceHeaderPropagation interface {
	Inject(ctx context.Context, carrier http.Header)
	Extract(ctx context.Context, carrier http.Header) context.Context
}

// NewTraceHeaderPropagation constructs a new trace header propagation.
func NewTraceHeaderPropagation() TraceHeaderPropagation {
	return &traceHeaderPropagation{propagation: propagation.TraceContext{}}
}
