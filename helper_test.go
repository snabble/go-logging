package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/snabble/go-logging/v2/tracex"
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

func Test_SetGoogle(t *testing.T) {
	require.NoError(t, SetGoogle("debug"))
	defer SetWithConfig("info", &DefaultLogConfig) // Reset to default

	tt := []struct {
		level   string
		message string
		logFn   func()
	}{
		{
			level:   "debug",
			message: "__debug__",
			logFn:   func() { Log.Debug("__debug__") },
		},
		{
			level:   "info",
			message: "__info__",
			logFn:   func() { Log.Info("__info__") },
		},
		{
			level:   "warning",
			message: "__warn__",
			logFn:   func() { Log.Warn("__warn__") },
		},
		{
			level:   "error",
			message: "__error__",
			logFn:   func() { Log.Error("__error__") },
		},
	}

	for _, tc := range tt {
		t.Run(tc.level, func(t *testing.T) {
			b := bytes.NewBuffer(nil)
			Log.Out = b

			tc.logFn()

			result := map[string]string{}
			require.NoError(t, json.Unmarshal(b.Bytes(), &result))

			assert.Equal(t, tc.message, result["message"])
			assert.Equal(t, tc.level, result["severity"])
			assert.NotEmpty(t, result["timestamp"])
		})
	}
}
