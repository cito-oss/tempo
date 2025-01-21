package tempo

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFunctionName(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		given := getFunctionName("ThisIsMyFunction")
		assert.Equal(t, "ThisIsMyFunction", given)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			getFunctionName(nil)
		})
	})

	t.Run("function", func(t *testing.T) {
		t.Parallel()

		given := getFunctionName(getFunctionName)
		assert.Equal(t, "getFunctionName", given)
	})

	t.Run("different package", func(t *testing.T) {
		t.Parallel()

		given := getFunctionName(math.Round)
		assert.Equal(t, "Round", given)
	})
}
