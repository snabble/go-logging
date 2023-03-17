package logging

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/snabble/go-logging/v2/tracex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LifecycleStart_AcceptsNil(t *testing.T) {
	assert.NotPanics(t, func() {
		LifecycleStart("app", nil)
	})
}

func Test_Call_UsesTrace(t *testing.T) {
	capture := capturingLogger(t)
	defer func() { _ = Set("info", true) }()
	provider := tracex.NewGlobalNoopTraceProvider("sampleApp", "v1.0.0")

	ctx, span := startSpan()
	defer span.End()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

	Call(req, &http.Response{}, time.Time{}, nil)

	require.NoError(t, provider.Shutdown(context.Background()))
	assert.NotContains(t, capture.String(), "00000000000000000000000000000000")
	assert.Contains(t, capture.String(), "trace")
	assert.Contains(t, capture.String(), "span")
}
