package tempo

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
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
		require.ErrorIs(t, err, ErrWorkflowExecute)

		temporalClient.AssertExpectations(t)
	})

	t.Run("fail to get workflow result", func(t *testing.T) {
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
		require.ErrorIs(t, err, ErrWorkflowResult)

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

		require.Equal(t, 2, cap(runner.limit))

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

func TestRunnerReport(t *testing.T) {
	t.Parallel()

	t.Run("simple success", func(t *testing.T) {
		t.Parallel()

		workflowRun := &mocks.WorkflowRun{}

		workflowRun.
			On("GetID").
			Return("abc")

		workflowRun.
			On("GetRunID").
			Return("123")

		workflowRun.
			On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				ptr := args[1].(*string)
				*ptr = "Hello World"
			}).
			Return(nil)

		temporalClient := &mocks.Client{}

		temporalClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(workflowRun, nil)

		temporalClient.
			On("DescribeWorkflowExecution", mock.Anything, "abc", "123").
			Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil)

		temporalClient.
			On("ListWorkflow", mock.Anything, mock.Anything).
			Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil)

		called := false

		opts := []Option{
			WithReporting(true),
			WithReportHandler(func(result *allure.Result) {
				called = true

				assert.Equal(t, "my_test", result.Name)
				assert.Equal(t, "my_id", result.FullName)
				assert.Equal(t, allure.Passed, result.Status)

				require.Len(t, result.Parameters, 2)

				assert.Equal(t, "input", result.Parameters[0].Name)
				assert.Equal(t, "World", result.Parameters[0].Value)

				assert.Equal(t, "output", result.Parameters[1].Name)
				assert.Equal(t, "Hello World", result.Parameters[1].Value)
			}),
		}

		var output string

		err := Run(temporalClient, "TestQueue", "my_id", NewPlan("my_test", "World"), &output, opts...)
		require.NoError(t, err)

		require.True(t, called)

		assert.Equal(t, "Hello World", output)
	})

	t.Run("simple fail", func(t *testing.T) {
		t.Parallel()

		workflowRun := &mocks.WorkflowRun{}

		workflowRun.
			On("GetID").
			Return("abc")

		workflowRun.
			On("GetRunID").
			Return("123")

		workflowRun.
			On("Get", mock.Anything, mock.Anything).
			Return(nil)

		temporalClient := &mocks.Client{}

		temporalClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(workflowRun, ErrWorkflowExecute)

		temporalClient.
			On("DescribeWorkflowExecution", mock.Anything, "abc", "123").
			Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil)

		temporalClient.
			On("ListWorkflow", mock.Anything, mock.Anything).
			Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil)

		ptrinput := 1234

		runner := NewRunner(temporalClient, "TestQueue", NewPlan("my_test", &ptrinput))

		called := false

		runner.SetReporting(true)
		runner.SetResultHandler(func(result *allure.Result) {
			called = true

			assert.Equal(t, "my_test", result.Name)
			assert.Equal(t, "TestPrefix@my_test", result.FullName)
			assert.Equal(t, allure.Failed, result.Status)

			require.Len(t, result.Parameters, 1)

			assert.Equal(t, "input", result.Parameters[0].Name)
			assert.InEpsilon(t, 1234, result.Parameters[0].Value, 0)
		})

		assert.True(t, runner.reporting)
		assert.NotNil(t, runner.handler)

		err := runner.Run("TestPrefix")
		require.Error(t, err)

		require.True(t, called)
	})

	t.Run("success with child workflow", func(t *testing.T) {
		t.Parallel()

		workflowRun := &mocks.WorkflowRun{}

		workflowRun.
			On("GetID").
			Return("root-id-abc")

		workflowRun.
			On("GetRunID").
			Return("root-run-id-123")

		workflowRun.
			On("Get", mock.Anything, mock.Anything).
			Return(nil)

		temporalClient := &mocks.Client{}

		temporalClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(workflowRun, nil)

		rootStep := allure.Step{
			Name: "root-step-1",
			Attachments: []*allure.Attachment{
				{
					Name:   reportReplaceFlag,
					Source: reportReplaceID("child-id-abc", "child-run-id-123"),
				},
			},
			Status: allure.Passed,
			StatusDetails: allure.StatusDetail{
				Message: "root message",
				Trace:   "root trace",
			},
			Start: 12345678,
			Stop:  12345678,
		}

		temporalClient.
			On("DescribeWorkflowExecution", mock.Anything, "root-id-abc", "root-run-id-123").
			Return(&workflowservice.DescribeWorkflowExecutionResponse{
				WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
					Memo: &common.Memo{
						Fields: map[string]*common.Payload{
							reportStepField: mustConvertToPayload(t, rootStep),
						},
					},
				},
			}, nil)

		childStep := allure.Step{
			Name:   "child-step-1",
			Status: allure.Passed,
			StatusDetails: allure.StatusDetail{
				Message: "child message",
				Trace:   "child trace",
			},
			Start: 987654,
			Stop:  987654,
		}

		temporalClient.
			On("ListWorkflow", mock.Anything, mock.Anything).
			Return(&workflowservice.ListWorkflowExecutionsResponse{
				Executions: []*workflowpb.WorkflowExecutionInfo{
					{
						Execution: &common.WorkflowExecution{
							WorkflowId: "child-id-abc",
							RunId:      "child-run-id-123",
						},
						Memo: &common.Memo{
							Fields: map[string]*common.Payload{
								reportStepField: mustConvertToPayload(t, childStep),
							},
						},
					},
				},
			}, nil)

		called := false

		opts := []Option{
			WithReporting(true),
			WithReportHandler(func(result *allure.Result) {
				called = true

				assert.Equal(t, "my_test", result.Name)
				assert.Equal(t, "my_id", result.FullName)
				assert.Equal(t, allure.Passed, result.Status)

				expected := &allure.Step{
					Name:   "root-step-1",
					Status: allure.Passed,
					StatusDetails: allure.StatusDetail{
						Message: "root message",
						Trace:   "root trace",
					},
					Start: 12345678,
					Stop:  12345678,
					Steps: []*allure.Step{
						{
							Name:   "child-step-1",
							Status: allure.Passed,
							StatusDetails: allure.StatusDetail{
								Message: "child message",
								Trace:   "child trace",
							},
							Start: 987654,
							Stop:  987654,
						},
					},
				}

				assert.Equal(t, expected, result.Steps[0])
			}),
		}

		err := Run(temporalClient, "TestQueue", "my_id", NewPlan("my_test", nil), nil, opts...)
		require.NoError(t, err)

		require.True(t, called)
	})
}

func TestRunnerReportSaving(t *testing.T) {
	// t.Parallel() // can't use parallel because the use of t.Setenv

	workflowRun := &mocks.WorkflowRun{}

	workflowRun.
		On("GetID").
		Return("abc")

	workflowRun.
		On("GetRunID").
		Return("123")

	workflowRun.
		On("Get", mock.Anything, mock.Anything).
		Return(nil)

	temporalClient := &mocks.Client{}

	temporalClient.
		On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(workflowRun, nil)

	temporalClient.
		On("DescribeWorkflowExecution", mock.Anything, "abc", "123").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil)

	temporalClient.
		On("ListWorkflow", mock.Anything, mock.Anything).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil)

	tmpdir := t.TempDir()

	t.Setenv("ALLURE_OUTPUT_PATH", tmpdir)

	runner := NewRunner(temporalClient, "TestQueue", NewPlan("my_test", nil))

	runner.SetReporting(true)

	assert.True(t, runner.reporting)

	err := runner.Run("TestPrefix")
	require.NoError(t, err)

	content := mustReadFirstFileContent(t, filepath.Join(tmpdir, "allure-results"))

	assert.Contains(t, content, `"name":"my_test"`)
	assert.Contains(t, content, `"fullName":"TestPrefix@my_test"`)
	assert.Contains(t, content, `"status":"passed"`)
	assert.Contains(t, content, `"statusDetails":{"message":"","trace":""}`)
}

func mustReadFirstFileContent(t *testing.T, tmpdir string) string {
	t.Helper()

	files, err := os.ReadDir(tmpdir)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(files), 1)

	filename := filepath.Join(tmpdir, files[0].Name())

	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	return string(data)
}

func mustConvertToPayload(t *testing.T, value any) *common.Payload {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(value)
	require.NoError(t, err)
	return payload
}
