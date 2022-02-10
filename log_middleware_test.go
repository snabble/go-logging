package logging

import (
	"bytes"
	"context"
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

	r, _ := http.NewRequest("GET", "https://www.example.org/foo", nil)

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
		var i []int
		i[100]++
	}))

	r, _ := http.NewRequest("GET", "http://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	data := logRecordFromBuffer(b)

	assert.Contains(t, data.Error, "Test_LogMiddleware_Panic.func1")
	assert.Contains(t, data.Error, "runtime error: index out of range")
	assert.Contains(t, data.Message, "ERROR ->GET /foo")
	assert.Contains(t, data.Level, "error")
}

func Test_LogMiddleware_Log_implicit200(t *testing.T) {
	a := assert.New(t)

	// given: a logger
	b := bytes.NewBuffer(nil)
	Log.Out = b

	// and a handler which gets an 200er code implicitly
	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	r, _ := http.NewRequest("GET", "http://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	data := logRecordFromBuffer(b)
	a.Equal("", data.Error)
	a.Equal("200 ->GET /foo", data.Message)
	a.Equal(200, data.ResponseStatus)
	a.Equal("info", data.Level)
}

func Test_LogMiddleware_Log_404(t *testing.T) {
	a := assert.New(t)

	// given: a logger
	b := bytes.NewBuffer(nil)
	Log.Out = b

	// and a handler which gets an 404er code explicitly
	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))

	r, _ := http.NewRequest("GET", "http://www.example.org/foo", nil)

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
