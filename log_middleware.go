package logging

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/snabble/go-utils/tracex"
)

type LogMiddleware struct {
	Next http.Handler
}

func NewLogMiddleware(next http.Handler) http.Handler {
	if Log.config.EnableTraces {
		return tracex.NewHandler(&LogMiddleware{Next: next}, "common")
	}
	return &LogMiddleware{Next: next}
}

func (mw *LogMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	defer func() {
		if rec := recover(); rec != nil {
			// See: https://pkg.go.dev/net/http#ErrAbortHandler
			if recErr, ok := rec.(error); ok && errors.Is(recErr, http.ErrAbortHandler) {
				AccessAborted(r, start)
				return
			}
			AccessError(r, start, fmt.Errorf("PANIC (%v): %v", identifyLogOrigin(), rec))
		}
	}()

	lrw := &logResponseWriter{ResponseWriter: w}
	mw.Next.ServeHTTP(lrw, r)

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
