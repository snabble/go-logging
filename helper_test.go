package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LifecycleStart_AcceptsNil(t *testing.T) {
	assert.NotPanics(t, func() {
		LifecycleStart("app", nil)
	})
}
