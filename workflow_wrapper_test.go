package tempo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestWorkflowWrapper(t *testing.T) {
	t.Parallel()

	t.Run("success with func(*T)", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := workflowWrapper[any, any]{
			name: "my_test",
			fn: func(myt *T) {
				assert.Equal(t, "my_test", myt.name)

				var given string
				err := myt.Task("myActivity", "world", &given)
				require.NoError(t, err)

				assert.Equal(t, "hello world", given)
			},
		}

		env.RegisterWorkflowWithOptions(wrapped.Function(), workflow.RegisterOptions{Name: wrapped.Name()})
		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("my_test")

		assert.True(t, called)
	})

	t.Run("success with func(*T, INPUT)", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := workflowWrapper[string, any]{
			name: "my_test_with_input",
			fnWithIn: func(myt *T, name string) {
				assert.Equal(t, "my_test_with_input", myt.name)
				assert.Equal(t, "world", name)

				var given string
				err := myt.Task("myActivity", name, &given)
				require.NoError(t, err)

				assert.Equal(t, "hello world", given)
			},
		}

		env.RegisterWorkflowWithOptions(wrapped.Function(), workflow.RegisterOptions{Name: wrapped.Name()})
		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("my_test_with_input", "world")

		assert.True(t, called)
	})

	t.Run("success with func(*T) OUTPUT", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := workflowWrapper[any, string]{
			name: "my_test_with_output",
			fnWithOut: func(myt *T) string {
				assert.Equal(t, "my_test_with_output", myt.name)

				var given string
				err := myt.Task("myActivity", "world", &given)
				require.NoError(t, err)

				assert.Equal(t, "hello world", given)

				return given
			},
		}

		env.RegisterWorkflowWithOptions(wrapped.Function(), workflow.RegisterOptions{Name: wrapped.Name()})
		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("my_test_with_output")

		var given string

		err := env.GetWorkflowResult(&given)
		require.NoError(t, err)

		assert.Equal(t, "hello world", given)

		assert.True(t, called)
	})

	t.Run("success with func(*T, INPUT) OUTPUT", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := workflowWrapper[string, string]{
			name: "my_test_with_input_and_output",
			fnWithInAndOut: func(myt *T, name string) string {
				assert.Equal(t, "my_test_with_input_and_output", myt.name)
				assert.Equal(t, "world", name)

				var given string
				err := myt.Task("myActivity", name, &given)
				require.NoError(t, err)

				assert.Equal(t, "hello world", given)

				return given
			},
		}

		env.RegisterWorkflowWithOptions(wrapped.Function(), workflow.RegisterOptions{Name: wrapped.Name()})
		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("my_test_with_input_and_output", "world")

		var given string

		err := env.GetWorkflowResult(&given)
		require.NoError(t, err)

		assert.Equal(t, "hello world", given)

		assert.True(t, called)
	})

	t.Run("errorf", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		wrapped := workflowWrapper[any, any]{
			name: "my_test",
			fn: func(myt *T) {
				myt.Errorf("something went wrong")
			},
		}

		env.RegisterWorkflowWithOptions(wrapped.Function(), workflow.RegisterOptions{Name: wrapped.Name()})

		env.ExecuteWorkflow("my_test")

		err := env.GetWorkflowError()
		require.Error(t, err)

		fail := &temporal.ApplicationError{}
		require.ErrorAs(t, err, &fail)

		assert.Equal(t, TestFailedErrorType, fail.Type())
		assert.ErrorContains(t, fail, "something went wrong")
	})

	t.Run("fail now", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		wrapped := workflowWrapper[any, any]{
			name: "my_test",
			fn: func(myt *T) {
				myt.FailNow()
			},
		}

		env.RegisterWorkflowWithOptions(wrapped.Function(), workflow.RegisterOptions{Name: wrapped.Name()})

		env.ExecuteWorkflow("my_test")

		err := env.GetWorkflowError()
		require.Error(t, err)

		fail := &temporal.ApplicationError{}
		require.ErrorAs(t, err, &fail)

		assert.Equal(t, TestFailedErrorType, fail.Type())
		assert.ErrorContains(t, fail, "test failed")
	})
}
