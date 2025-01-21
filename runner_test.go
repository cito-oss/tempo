package tempo

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

func TestRunnerRun(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		workflowRun := &mocks.WorkflowRun{}

		workflowRun.
			On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {}).
			Return(nil)

		temporalClient := &mocks.Client{}

		temporalClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, "MyTestFunction", "SomeInput").
			Run(func(args mock.Arguments) {
				assert.Equal(t, "TestPrefix@MyTestFunction", args[1].(client.StartWorkflowOptions).ID)
				assert.Equal(t, "TestQueue", args[1].(client.StartWorkflowOptions).TaskQueue)
			}).
			Return(workflowRun, nil).
			Once()

		runner := NewRunner(temporalClient, "TestQueue", NewPlan("MyTestFunction", "SomeInput"))

		err := runner.Run("TestPrefix")
		require.NoError(t, err)

		workflowRun.AssertExpectations(t)
		temporalClient.AssertExpectations(t)
	})

	t.Run("fail to execute workflow", func(t *testing.T) {
		t.Parallel()

		temporalClient := &mocks.Client{}

		temporalClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("temporal error")).
			Once()

		runner := NewRunner(temporalClient, "TestQueue", NewPlan("MyTestFunction", nil))

		err := runner.Run("TestPrefix")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWorkflowExecute)

		temporalClient.AssertExpectations(t)
	})

	t.Run("fail to get workflow run", func(t *testing.T) {
		t.Parallel()

		workflowRun := &mocks.WorkflowRun{}

		workflowRun.
			On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {}).
			Return(errors.New("temporal error"))

		temporalClient := &mocks.Client{}

		temporalClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(workflowRun, nil).
			Once()

		runner := NewRunner(temporalClient, "TestQueue", NewPlan("MyTestFunction", nil))

		err := runner.Run("TestPrefix")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWorkflow)

		temporalClient.AssertExpectations(t)
		workflowRun.AssertExpectations(t)
	})

	t.Run("runner with limit", func(t *testing.T) {
		t.Parallel()

		inflight := atomic.Int32{}

		workflowRun := &mocks.WorkflowRun{}

		workflowRun.
			On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				// asserts that it nevers exceeds the limit
				assert.LessOrEqual(t, inflight.Load(), int32(2))

				inflight.Add(-1)
			}).
			Return(nil)

		temporalClient := &mocks.Client{}

		temporalClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				inflight.Add(1)

				time.Sleep(10 * time.Millisecond)
			}).
			Return(workflowRun, nil)

		runner := NewRunner(temporalClient, "TestQueue",
			NewPlan("MyTestFunction", "Input1"),
			NewPlan("MyTestFunction", "Input2"),
			NewPlan("MyTestFunction", "Input3"),
			NewPlan("MyTestFunction", "Input4"),
			NewPlan("MyTestFunction", "Input5"),
		)

		runner.SetLimit(2)

		require.Equal(t, cap(runner.limit), 2)

		err := runner.Run("TestPrefix")
		require.NoError(t, err)
	})

	t.Run("runner with negative limit", func(t *testing.T) {
		t.Parallel()

		runner := &Runner{}
		runner.SetLimit(2)
		require.NotNil(t, runner.limit)

		runner.SetLimit(-1)
		require.Nil(t, runner.limit)
	})

	t.Run("change limit while running", func(t *testing.T) {
		t.Parallel()

		runner := &Runner{}
		runner.SetLimit(42)

		runner.limit <- struct{}{}

		require.Panics(t, func() {
			runner.SetLimit(1)
		})
	})
}
