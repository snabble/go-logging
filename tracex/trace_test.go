package tracex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test_BackgroundContextWithSpan(t *testing.T) {
	originalContextWithCancel, cancel := context.WithCancel(context.Background())
	originalCtx, originalSpan := noop.NewTracerProvider().Tracer("").Start(originalContextWithCancel, "spanName")
	newCtxWithSameSpan := BackgroundContextWithSpan(originalCtx)
	trace.SpanContextFromContext(newCtxWithSameSpan).Equal(originalSpan.SpanContext())

	cancel()
	assert.NotNil(t, originalCtx.Err())

	assert.Nil(t, newCtxWithSameSpan.Err())
}
