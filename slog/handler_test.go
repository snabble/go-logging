package slog

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snabble/go-logging/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Slog_Set(t *testing.T) {
	a := assert.New(t)

	// given: an error logger in text format
	logging.Set("error", true)
	defer logging.Set("info", false)
	logging.Log.Formatter.(*logrus.TextFormatter).DisableColors = true
	b := bytes.NewBuffer(nil)
	logging.Log.Out = b

	slog := New()

	// when: I log something
	slog.Info("should be ignored ..")
	slog.With("foo", "bar").Error("oops")

	// then: only the error text is contained, and it is text formatted
	a.Regexp(`^time.* level\=error msg\=oops foo\=bar.*`, b.String())
}

func Test_Slog_WithError(t *testing.T) {
	a := assert.New(t)

	// given: an logger in text format
	logging.Set("info", true)
	defer logging.Set("info", false)
	logging.Log.Formatter.(*logrus.TextFormatter).DisableColors = true
	b := bytes.NewBuffer(nil)
	logging.Log.Out = b

	slog := New()

	err := func() error {
		return fmt.Errorf("found an error: %w", errors.New("an error occurred"))
	}()
	slog.Error("oops", "error", err)

	a.Regexp(`^time.* level\=error msg\=oops error\="found an error: an error occurred"`, b.String())
}
