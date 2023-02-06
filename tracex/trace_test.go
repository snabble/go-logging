package tracex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func Test_BackgroundContextWithSpan(t *testing.T) {
	originalContextWithCancel, cancel := context.WithCancel(context.Background())
	originalCtx, originalSpan := trace.NewNoopTracerProvider().Tracer("").Start(originalContextWithCancel, "spanName")
	newCtxWithSameSpan := BackgroundContextWithSpan(originalCtx)
	trace.SpanContextFromContext(newCtxWithSameSpan).Equal(originalSpan.SpanContext())

	cancel()
	assert.NotNil(t, originalCtx.Err())

	assert.Nil(t, newCtxWithSameSpan.Err())
}
