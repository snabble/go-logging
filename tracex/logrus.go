package tracex

import (
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
)

// NewLogrusHook creates a new logrus hook for otel with log levels from info and up.
// This is used by our go-logging library.
func NewLogrusHook() logrus.Hook {
	return otellogrus.NewHook("", otellogrus.WithLevels([]logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}))
}
