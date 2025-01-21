package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTest(t *testing.T) {
	t.Parallel()

	t.Run("new test", func(t *testing.T) {
		t.Parallel()

		test := NewTest(Example)
		assert.Equal(t, "Example", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[any, any]).fn)
	})

	t.Run("new test with input", func(t *testing.T) {
		t.Parallel()

		test := NewTestWithInput(ExampleWithInput)
		assert.Equal(t, "ExampleWithInput", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[string, any]).fnWithIn)
	})

	t.Run("new test with output", func(t *testing.T) {
		t.Parallel()

		test := NewTestWithOutput(ExampleWithOutput)
		assert.Equal(t, "ExampleWithOutput", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[any, string]).fnWithOut)
	})

	t.Run("new test with input and output", func(t *testing.T) {
		t.Parallel()

		test := NewTestWithInputAndOutput(ExampleWithInputAndOutput)
		assert.Equal(t, "ExampleWithInputAndOutput", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[string, string]).fnWithInAndOut)
	})
}

func Example(t *T) {}

func ExampleWithInput(t *T, input string) {}

func ExampleWithOutput(t *T) string {
	return ""
}

func ExampleWithInputAndOutput(t *T, input string) string {
	return ""
}
