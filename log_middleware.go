package logging

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type LogMiddlewareWithSpan struct {
	Next http.Handler
}

type LogMiddleware struct {
	Next http.Handler
}

func NewLogMiddleware(next http.Handler) http.Handler {
	if Log.config.EnableTraces {
		return otelhttp.NewHandler(&LogMiddlewareWithSpan{Next: next}, "common")
	}
	return &LogMiddleware{Next: next}
}

func (mw *LogMiddlewareWithSpan) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//span := trace.SpanFromContext(r.Context())
	//defer span.End()

	serveHTTP(w, r, mw.Next)
}

func (mw *LogMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serveHTTP(w, r, mw.Next)
}

func serveHTTP(w http.ResponseWriter, r *http.Request, mw http.Handler) {
	start := time.Now()

	defer func() {
		if rec := recover(); rec != nil {
			AccessError(r, start, fmt.Errorf("PANIC (%v): %v", identifyLogOrigin(), rec))
		}
	}()

	lrw := &logResponseWriter{ResponseWriter: w}
	mw.ServeHTTP(lrw, r)

	Access(r, start, lrw.statusCode)
}

// identifyLogOrigin returns the location, where a panic was raised
// in the form package/subpackage.method:line
func identifyLogOrigin() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}

type logResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *logResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = 200
	}
	return lrw.ResponseWriter.Write(b)
}

func (lrw *logResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}
