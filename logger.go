package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ScopeField          = "scope"
	ProjectField        = "project"
	ShopField           = "shop"
	CheckoutField       = "checkout"
	OrderField          = "order"
	CheckoutDeviceField = "checkoutDevice"
	DurationField       = "duration"
	FlakyField          = "flaky"
	TypeField           = "type"

	TypeDeprecation = "deprecation"
	TypeCall        = "call"
	TypeAccess      = "access"
	TypeApplication = "application"
	TypeLifecycle   = "lifecycle"
	TypeCacheinfo   = "cacheinfo"
)

type Identifiable interface {
	LogIdentity() map[string]any
}

var Log *Logger

type Logger struct {
	*logrus.Logger
	config *LogConfig
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

func (logger *Logger) With(o Identifiable) *Entry {
	return NewEntry(logger).With(o)
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

func (logger *Logger) WithProject(projectID string) *Entry {
	return NewEntry(logger).WithField(ProjectField, projectID)
}

func (logger *Logger) WithShop(shopID string) *Entry {
	return NewEntry(logger).WithField(ShopField, shopID)
}

func (logger *Logger) WithCheckout(checkoutID string) *Entry {
	return NewEntry(logger).WithField(CheckoutField, checkoutID)
}

func (logger *Logger) WithCheckoutDevice(checkoutDeviceID string) *Entry {
	return NewEntry(logger).WithField(CheckoutDeviceField, checkoutDeviceID)
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
	withError := entry.WithField(logrus.ErrorKey, err)
	stacktrace := ExtractStacktrace(err)
	if stacktrace != nil {
		return withError.WithField("stacktrace", fmt.Sprintf("%+v", stacktrace))
	}
	return withError
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

func (entry *Entry) With(o Identifiable) *Entry {
	return wrapEntry(entry.Entry.WithFields(o.LogIdentity()))
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

func (entry *Entry) WithProject(projectID string) *Entry {
	return entry.WithField(ProjectField, projectID)
}

func (entry *Entry) WithShop(shop string) *Entry {
	return entry.WithField(ShopField, shop)
}

func (entry *Entry) WithCheckout(checkoutID string) *Entry {
	return entry.WithField(CheckoutField, checkoutID)
}

func (entry *Entry) WithCheckoutDevice(checkoutDeviceID string) *Entry {
	return entry.WithField(CheckoutDeviceField, checkoutDeviceID)
}

func (entry *Entry) WithOrder(orderID string) *Entry {
	return entry.WithField(OrderField, orderID)
}

// Deprecation marks the log entry with type "deprecation". This is used for log entries, that are logged during a
// feature change. Logs with "deprecation" types should not be considered as critical and should only occur temporarily
// during the transition.
func (entry *Entry) Deprecation() *Entry {
	return entry.WithField(TypeField, TypeDeprecation)
}

func wrapEntry(logrusEntry *logrus.Entry) *Entry {
	return &Entry{Entry: logrusEntry}
}
