package logging

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"github.com/snabble/go-logging/v2/tracex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func Test_NewGlobalNoopTraceProvider_SetsTheGlobalProvider(t *testing.T) {
	capture := capturingLogger(t)
	defer func() { _ = Set("info", true) }()

	provider := tracex.NewGlobalNoopTraceProvider("sampleApp", "v1.0.0")

	ctx, span := startSpan()
	defer span.End()
	Log.WithContext(ctx).Info("meee")

	require.NoError(t, provider.Shutdown(context.Background()))
	assert.NotContains(t, capture.String(), "00000000000000000000000000000000")
	assert.Contains(t, capture.String(), "trace")
	assert.Contains(t, capture.String(), "span")
}

func Test_NewGlobalNoopTraceProvider_DoesNotTraceIfNotActivated(t *testing.T) {
	capture := capturingLogger(t)
	defer func() { _ = Set("info", true) }()

	ctx, span := startSpan()
	defer span.End()
	Log.WithContext(ctx).Info("meee")

	assert.NotContains(t, capture.String(), "trace")
	assert.NotContains(t, capture.String(), "span")
}

func startSpan() (context.Context, trace.Span) {
	return otel.Tracer("global").Start(context.Background(), "test")
}

func capturingLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	capture := &bytes.Buffer{}
	err := SetWithConfig("info", &LogConfig{EnableTraces: true, EnableTextLogging: false})
	require.NoError(t, err)
	Log.Hooks.Add(&writer.Hook{Writer: capture, LogLevels: logrus.AllLevels})
	return capture
}
