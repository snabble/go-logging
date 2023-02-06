package tracex

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

type tracePropagation struct {
	propagation propagation.TraceContext
}

func (t *tracePropagation) Inject(ctx context.Context, carrier map[string]string) {
	t.propagation.Inject(ctx, propagation.MapCarrier(carrier))
}

func (t *tracePropagation) Extract(ctx context.Context, carrier map[string]string) context.Context {
	return t.propagation.Extract(ctx, propagation.MapCarrier(carrier))
}

func (t *tracePropagation) Fields() []string {
	return t.propagation.Fields()
}

type TracePropagation interface {
	Inject(ctx context.Context, carrier map[string]string)
	Extract(ctx context.Context, carrier map[string]string) context.Context
}

// NewTracePropagation constructs a new trace propagation.
func NewTracePropagation() TracePropagation {
	return &tracePropagation{propagation: propagation.TraceContext{}}
}
