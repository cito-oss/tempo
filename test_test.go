package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTest(t *testing.T) {
	t.Parallel()

	t.Run("new test", func(t *testing.T) {
		t.Parallel()

		test := NewTest(MyWorkflow)
		assert.Equal(t, "MyWorkflow", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[any, any]).fn)
	})

	t.Run("new test with input", func(t *testing.T) {
		t.Parallel()

		test := NewTestWithInput(MyWorkflowWithInput)
		assert.Equal(t, "MyWorkflowWithInput", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[string, any]).fnWithIn)
	})

	t.Run("new test with output", func(t *testing.T) {
		t.Parallel()

		test := NewTestWithOutput(MyWorkflowWithOutput)
		assert.Equal(t, "MyWorkflowWithOutput", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[any, string]).fnWithOut)
	})

	t.Run("new test with input and output", func(t *testing.T) {
		t.Parallel()

		test := NewTestWithInputAndOutput(MyWorkflowWithInputAndOutput)
		assert.Equal(t, "MyWorkflowWithInputAndOutput", test.Name())
		assert.NotNil(t, test.(*workflowWrapper[string, string]).fnWithInAndOut)
	})
}

func MyWorkflow(t *T) {}

func MyWorkflowWithInput(t *T, input string) {}

func MyWorkflowWithOutput(t *T) string {
	return ""
}

func MyWorkflowWithInputAndOutput(t *T, input string) string {
	return ""
}
