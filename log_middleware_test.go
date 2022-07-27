package logging

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

const (
	fallBackForSpan  = "0000000000000000"
	fallBackForTrace = "00000000000000000000000000000000"
)

func Test_LogMiddleware_GeneratesTracesAndSubSpans(t *testing.T) {
	tp := initTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	err := SetWithConfig("info", &LogConfig{EnableTraces: true, EnableTextLogging: false})
	require.NoError(t, err)

	b := bytes.NewBuffer(nil)
	Log.Out = b

	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, childSpan := otel.Tracer("").Start(r.Context(), "child")
		defer childSpan.End()
		Log.Logger.WithContext(ctx).Info("That's awesome!")
		w.WriteHeader(200)
	}))

	r, _ := http.NewRequest(http.MethodGet, "https://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	logRecords := logRecordsFromBuffer(b)
	firstRecord := logRecords[0]
	assert.Contains(t, firstRecord.Message, "That's awesome!")
	assert.Contains(t, firstRecord.Level, "info")
	assertHasTraceAndSpan(t, firstRecord)

	secondRecord := logRecords[1]
	assert.Contains(t, secondRecord.Message, "200 ->GET /foo")
	assert.Contains(t, secondRecord.Level, "info")
	assertHasTraceAndSpan(t, secondRecord)

	assert.Equal(t, firstRecord.TraceID, secondRecord.TraceID)
	assert.NotEqual(t, firstRecord.SpanID, secondRecord.SpanID)
}

func assertHasTraceAndSpan(t *testing.T, record logRecord) {
	t.Helper()
	assert.Len(t, record.SpanID, 16)
	assert.NotEqual(t, fallBackForSpan, record.SpanID)

	assert.Len(t, record.TraceID, 32)
	assert.NotEqual(t, fallBackForTrace, record.TraceID)
}

func Test_LogMiddleware_Panic(t *testing.T) {
	_ = SetWithConfig("info", &LogConfig{EnableTraces: false, EnableTextLogging: false})

	b := bytes.NewBuffer(nil)
	Log.Out = b

	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("some"))
	}))

	r, _ := http.NewRequest(http.MethodGet, "http://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	data := logRecordFromBuffer(b)

	assert.Contains(t, data.Error, "Test_LogMiddleware_Panic.func1")
	assert.Contains(t, data.Error, "some")
	assert.Equal(t, data.Message, "ERROR ->GET /foo")
	assert.Equal(t, data.Level, "error")
}

func Test_LogMiddleware_Panic_ErrAbortHandler(t *testing.T) {
	_ = SetWithConfig("info", &LogConfig{EnableTraces: false, EnableTextLogging: false})

	b := bytes.NewBuffer(nil)
	Log.Out = b

	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	r, _ := http.NewRequest(http.MethodGet, "http://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	data := logRecordFromBuffer(b)

	assert.Equal(t, data.Message, "ABORTED ->GET /foo")
	assert.Equal(t, data.Level, "info")
}

func Test_LogMiddleware_Log_implicit200(t *testing.T) {
	a := assert.New(t)

	// given: a logger
	b := bytes.NewBuffer(nil)
	Log.Out = b

	// and a handler which gets a 200er code implicitly
	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	r, _ := http.NewRequest(http.MethodGet, "http://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	data := logRecordFromBuffer(b)
	a.Equal("", data.Error)
	a.Equal("200 ->GET /foo", data.Message)
	a.Equal(200, data.ResponseStatus)
	a.Equal("info", data.Level)
}

func Test_LogMiddleware_Log_HealthRequestAreNotLogged(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		containsLogMsg bool
	}{
		{"health on base path", http.MethodGet, "http://www.example.org/health", false},
		{"health on sub path", http.MethodGet, "http://www.example.org/sub/health", false},
		{"post to health", http.MethodPost, "http://www.example.org/sub/health", true},
		{"health in query param", http.MethodGet, "http://www.example.org/sub?health", true},
		{"health somewhere in path", http.MethodGet, "http://www.example.org/health/more", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logBuffer := bytes.NewBuffer(nil)
			Log.Out = logBuffer

			lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("OK++"))
			}))

			r, _ := http.NewRequest(test.method, test.url, nil)

			lm.ServeHTTP(httptest.NewRecorder(), r)
			assert.Equal(t, test.containsLogMsg, logBuffer.Len() > 0)
		})
	}
}

func Test_LogMiddleware_Log_404(t *testing.T) {
	a := assert.New(t)

	// given: a logger
	b := bytes.NewBuffer(nil)
	Log.Out = b

	// and a handler which gets a 404 code explicitly
	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))

	r, _ := http.NewRequest(http.MethodGet, "http://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	data := logRecordFromBuffer(b)
	a.Equal("", data.Error)
	a.Equal("404 ->GET /foo", data.Message)
	a.Equal(404, data.ResponseStatus)
	a.Equal("warning", data.Level)
}

func initTracer() *sdktrace.TracerProvider {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String("logging-test"))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}
