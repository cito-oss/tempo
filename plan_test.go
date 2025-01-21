package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPlan(t *testing.T) {
	t.Parallel()

	t.Run("string plan", func(t *testing.T) {
		t.Parallel()

		got := NewPlan("argument 1", "argument 2")

		assert.Equal(t, "argument 1", got.function)
		assert.Equal(t, "argument 1", got.name)
		assert.Equal(t, "argument 2", got.input)

		assert.Equal(t, "argument 1", got.Name())
		assert.Equal(t, "argument 2", got.Input())
	})

	t.Run("function plan", func(t *testing.T) {
		t.Parallel()

		got := NewPlan(ExampleWithInputAndOutput, "example argument")

		assert.IsType(t, ExampleWithInputAndOutput, got.function)
		assert.IsType(t, "ExampleWithInputAndOutput", got.name)
		assert.Equal(t, "example argument", got.input)

		assert.Equal(t, "ExampleWithInputAndOutput", got.Name())
		assert.Equal(t, "example argument", got.Input())
	})
}
