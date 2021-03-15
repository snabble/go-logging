package logging

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ScopeField    = "scope"
	ProjectField  = "project"
	CheckoutField = "checkout"
	OrderField    = "order"
	DurationField = "duration"
)

var Log *Logger

type Logger struct {
	*logrus.Logger
}

func (logger *Logger) WithError(err error) *Entry {
	return NewEntry(logger).WithError(err)
}

func (logger *Logger) WithContext(ctx context.Context) *Entry {
	return NewEntry(logger).WithContext(ctx)
}

func (logger *Logger) WithField(key string, value interface{}) *Entry {
	return NewEntry(logger).WithField(key, value)
}

func (logger *Logger) WithFields(fields logrus.Fields) *Entry {
	return NewEntry(logger).WithFields(fields)
}

func (logger *Logger) WithTime(t time.Time) *Entry {
	return NewEntry(logger).WithTime(t)
}

func (logger *Logger) WithDuration(d time.Duration) *Entry {
	return NewEntry(logger).WithDuration(d)
}

func (logger *Logger) WithScope(scope string) *Entry {
	return NewEntry(logger).WithScope(scope)
}

func (logger *Logger) WithProject(project string) *Entry {
	return NewEntry(logger).WithField(ProjectField, project)
}

func (logger *Logger) WithCheckout(checkout string) *Entry {
	return NewEntry(logger).WithField(CheckoutField, checkout)
}

func (logger *Logger) WithOrder(order string) *Entry {
	return NewEntry(logger).WithField(OrderField, order)
}

type Entry struct {
	*logrus.Entry
}

func NewEntry(logger *Logger) *Entry {
	return wrapEntry(logrus.NewEntry(logger.Logger))
}

func (entry *Entry) WithError(err error) *Entry {
	return entry.WithField(logrus.ErrorKey, err)
}

func (entry *Entry) WithContext(ctx context.Context) *Entry {
	return wrapEntry(entry.Entry.WithContext(ctx))
}

func (entry *Entry) WithField(key string, value interface{}) *Entry {
	return entry.WithFields(logrus.Fields{key: value})
}

func (entry *Entry) WithFields(fields logrus.Fields) *Entry {
	return wrapEntry(entry.Entry.WithFields(fields))
}

func (entry *Entry) WithTime(t time.Time) *Entry {
	return wrapEntry(entry.Entry.WithTime(t))
}

func (entry *Entry) WithDuration(d time.Duration) *Entry {
	return entry.WithField(DurationField, d.Nanoseconds()/1000000)
}

func (entry *Entry) WithScope(scope string) *Entry {
	return entry.WithField(ScopeField, scope)
}

func (entry *Entry) WithProject(project string) *Entry {
	return entry.WithField(ProjectField, project)
}

func (entry *Entry) WithCheckout(checkout string) *Entry {
	return entry.WithField(CheckoutField, checkout)
}

func (entry *Entry) WithOrder(order string) *Entry {
	return entry.WithField(OrderField, order)
}

func wrapEntry(logrusEntry *logrus.Entry) *Entry {
	return &Entry{Entry: logrusEntry}
}
