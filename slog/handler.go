package slog

import (
	"context"
	"github.com/snabble/go-logging/v2"

	"log/slog"

	"github.com/sirupsen/logrus"
)

var logLevelsFromSlog = map[slog.Level]logrus.Level{
	slog.LevelDebug: logrus.DebugLevel,
	slog.LevelInfo:  logrus.InfoLevel,
	slog.LevelWarn:  logrus.WarnLevel,
	slog.LevelError: logrus.ErrorLevel,
}

var logLevelsToSlog = map[logrus.Level]slog.Level{
	logrus.DebugLevel: slog.LevelDebug,
	logrus.InfoLevel:  slog.LevelInfo,
	logrus.WarnLevel:  slog.LevelWarn,
	logrus.ErrorLevel: slog.LevelError,
}

type Option struct {
	Level           slog.Level
	Logger          *logging.Logger
	AttrFromContext []func(ctx context.Context) []slog.Attr
	AddSource       bool
	ReplaceAttr     func(groups []string, a slog.Attr) slog.Attr
}

func New() *slog.Logger {
	return slog.New(
		Option{
			Level:  logLevelsToSlog[logging.Log.GetLevel()],
			Logger: logging.Log,
		}.newLogrusHandler())
}

func (o Option) newLogrusHandler() slog.Handler {
	if o.AttrFromContext == nil {
		o.AttrFromContext = []func(ctx context.Context) []slog.Attr{}
	}

	return &LogrusHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

type LogrusHandler struct {
	option Option
	attrs  []slog.Attr
	groups []string
}

func (h *LogrusHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *LogrusHandler) Handle(ctx context.Context, record slog.Record) error {
	level := logLevelsFromSlog[record.Level]
	fromContext := contextExtractor(ctx, h.option.AttrFromContext)
	args := convert(h.option.AddSource, h.option.ReplaceAttr, append(h.attrs, fromContext...), h.groups, &record)

	logging.NewEntry(h.option.Logger).
		WithContext(ctx).
		WithTime(record.Time).
		WithFields(args).
		Log(level, record.Message)

	return nil
}

func (h *LogrusHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogrusHandler{
		option: h.option,
		attrs:  appendAttrsToGroup(h.groups, h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *LogrusHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &LogrusHandler{
		option: h.option,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}
