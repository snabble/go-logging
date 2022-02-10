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

func Test_LogMiddleware_GeneratesTraces(t *testing.T) {
	tp := initTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	err := SetWithConfig("info", &LogConfig{EnableTraces: true, EnableTextLogging: false})
	require.NoError(t, err)

	b := bytes.NewBuffer(nil)
	Log.Out = b

	lm := NewLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//ctx, childSpan := otel.Tracer("").Start(r.Context(), "child")
		//defer childSpan.End()
		//Log.Logger.WithContext(ctx).Info("That's awesome!")
		w.WriteHeader(200)
	}))

	r, _ := http.NewRequest("GET", "https://www.example.org/foo", nil)

	lm.ServeHTTP(httptest.NewRecorder(), r)

	data := logRecordFromBuffer(b)

	assert.Contains(t, data.Message, "200 ->GET /foo")
	assert.Contains(t, data.Level, "info")
	assert.Len(t, data.SpanID, 16)
	assert.NotEqual(t, "0000000000000000", data.SpanID)
	assert.Len(t, data.TraceID, 32)
	assert.NotEqual(t, "00000000000000000000000000000000", data.TraceID)
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
