package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestFailedError(t *testing.T) {
	t.Parallel()

	t.Run("single msg", func(t *testing.T) {
		t.Parallel()

		err := NewTestFailedError([]string{"message 1"})
		assert.Equal(t, "message 1", err.Error())
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		err := NewTestFailedError(nil)
		assert.Equal(t, "test failed", err.Error())
	})
}
