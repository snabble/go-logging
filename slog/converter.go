package sloglogrus

import (
	"log/slog"
)

var SourceKey = "source"
var ErrorKeys = []string{"error", "err"}

type converter func(addSource bool, replaceAttr func(groups []string, a slog.Attr) slog.Attr, loggerAttr []slog.Attr, groups []string, record *slog.Record) map[string]any

func defaultConverter(addSource bool, replaceAttr func(groups []string, a slog.Attr) slog.Attr, loggerAttr []slog.Attr, groups []string, record *slog.Record) map[string]any {
	attrs := appendRecordAttrsToAttrs(loggerAttr, groups, record)

	attrs = replaceError(attrs, ErrorKeys...)
	if addSource {
		attrs = append(attrs, source(SourceKey, record))
	}
	attrs = replaceAttrs(replaceAttr, []string{}, attrs...)
	attrs = removeEmptyAttrs(attrs)

	output := attrsToMap(attrs...)

	return output
}
