package audit

import (
	"context"

	"github.com/snabble/go-logging/v2"
)

type Entry struct {
	entry *logging.Entry
	event string
}

func Event(ctx context.Context, event string) *Entry {
	return &Entry{
		entry: logging.Log.
			WithContext(ctx).
			WithField(logging.TypeField, "audit"),
		event: event,
	}
}

func (e *Entry) WithTransaction(txnID string) *Entry {
	e.entry = e.entry.WithTransaction(txnID)
	return e
}

func (e *Entry) WithProject(projectID string) *Entry {
	e.entry = e.entry.WithProject(projectID)
	return e
}

func (e *Entry) WithShop(shopID string) *Entry {
	e.entry = e.entry.WithShop(shopID)
	return e
}

func (e *Entry) Send() {
	e.entry.Info(e.event)
}
