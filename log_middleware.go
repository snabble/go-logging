package logging

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/snabble/go-logging/v2/tracex"
)

type LogMiddlewareConfig struct {
	// SkipSuccessfulRequestsMatching is a list of go reqular
	// expressions, the access log is skipped if for request where the
	// request path matches the expression.
	SkipSuccessfulRequestsMatching []string
}

type LogMiddleware struct {
	Next http.Handler

	skipCache []*regexp.Regexp
}

func NewLogMiddleware(next http.Handler) http.Handler {
	handler, _ := AddLogMiddleware(next, LogMiddlewareConfig{})
	return handler
}

func AddLogMiddleware(next http.Handler, cfg LogMiddlewareConfig) (http.Handler, error) {
	skipCache, err := buildSkipCache(cfg)
	if err != nil {
		return nil, err
	}

	middleware := &LogMiddleware{
		Next:      next,
		skipCache: skipCache,
	}

	if Log.config.EnableTraces {
		return tracex.NewHandler(middleware, "common"), nil
	}

	return middleware, nil
}

func buildSkipCache(cfg LogMiddlewareConfig) ([]*regexp.Regexp, error) {
	var skipCache []*regexp.Regexp

	for _, expr := range cfg.SkipSuccessfulRequestsMatching {
		compiled, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid matcher: '%s': %w", expr, err)
		}

		skipCache = append(skipCache, compiled)
	}

	return skipCache, nil
}

func (mw *LogMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	lrw := &logResponseWriter{ResponseWriter: w}

	defer func() {
		if rec := recover(); rec != nil {
			lrw.WriteHeader(http.StatusInternalServerError)
			// See: https://pkg.go.dev/net/http#ErrAbortHandler
			if recErr, ok := rec.(error); ok && errors.Is(recErr, http.ErrAbortHandler) {
				AccessAborted(r, start)
				return
			}
			AccessError(r, start, fmt.Errorf("PANIC (%v): %v", identifyLogOrigin(), rec), debug.Stack())
		}
	}()

	mw.Next.ServeHTTP(lrw, r)

	level := logrus.InfoLevel
	if mw.isSkipped(r.URL.Path) {
		level = logrus.DebugLevel
	}

	access(level, r, start, lrw.statusCode)
}

func (mw *LogMiddleware) isSkipped(path string) bool {
	for _, exp := range mw.skipCache {
		if exp.MatchString(path) {
			return true
		}
	}
	return false
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
