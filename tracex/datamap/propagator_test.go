package datamap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	traceID      = "11000000000000000000000000000000"
	spanID       = "0100000000000000"
	traceAndSpan = "00-11000000000000000000000000000000-0100000000000000-00"
)

var (
	spanContext = trace.NewSpanContext(
		trace.SpanContextConfig{
			TraceID: trace.TraceID{0x11},
			SpanID:  trace.SpanID{0x01},
		})
)

func Test_Inject_DataMapIsFilledWithTrace(t *testing.T) {
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), spanContext)
	ctx, _ = traceProvider().
		Tracer("__traced_app__").Start(ctx, "test")

	inject := propagator().Inject(ctx)

	container := dataMap(inject)
	assert.Equal(t, container.Keys(), []string{"traceparent"})
	assert.Equal(t, traceAndSpan, container.Get("traceparent"))
}

type someDataMapContainer struct {
	dm map[string]string
}

func (s someDataMapContainer) GetDataMap() map[string]string {
	return s.dm
}

func Test_Extract_PopulatesContext(t *testing.T) {
	workItem := someDataMapContainer{dm: map[string]string{"__tracing__traceparent": traceAndSpan}}

	ctx := context.Background()
	extractedCtx := propagator().Extract(ctx, workItem)

	span := trace.SpanContextFromContext(extractedCtx)
	assert.True(t, span.IsRemote())
	assert.Equal(t, traceID, span.TraceID().String())
	assert.Equal(t, spanID, span.SpanID().String())
}

func Test_Extract_DoesNotFailOnEmptyDataMap(t *testing.T) {
	workItem := someDataMapContainer{dm: map[string]string{}}

	ctx := context.Background()
	extractedCtx := propagator().Extract(ctx, workItem)

	span := trace.SpanContextFromContext(extractedCtx)
	assert.False(t, span.IsRemote())
	assert.False(t, span.IsValid())
}

func propagator() *Propagator {
	return NewPropagator()
}

func traceProvider() trace.TracerProvider {
	return noop.NewTracerProvider()
}
