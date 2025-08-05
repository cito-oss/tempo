package tempo

import (
	"context"
	"errors"
	"testing"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestWorkflowWrapper(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := Workflow[any, any]{
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

	t.Run("success with input", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := Workflow[string, any]{
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

	t.Run("success with output", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := Workflow[any, string]{
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

	t.Run("success with input and output", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		wrapped := Workflow[string, string]{
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

		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()

		wrapped := Workflow[any, any]{
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

		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()

		wrapped := Workflow[any, any]{
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

func TestWorkflowWrapperReport(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	wrapped := Workflow[any, any]{
		name: "my_test",
		fn: func(myt *T) {
			myt.Run("sub_test", func(myt *T) {
				myt.Run("another_sub_test", func(myt *T) {
					myt.Go(func(myt *T) {
						myt.FailNow()
					})

					myt.Go(func(myt *T) {
						// do nothing
					})
				})
			})

			myt.Run("other_test", func(myt *T) {
				myt.Run("other_sub_test", func(myt *T) {
					myt.Go(func(myt *T) {
						// do nothing
					})
				})
			})
		},
	}

	env.RegisterWorkflowWithOptions(wrapped.Function(), workflow.RegisterOptions{Name: wrapped.Name()})

	env.
		OnUpsertMemo(mock.Anything).
		Run(func(args mock.Arguments) {
			require.NotEmpty(t, args)

			field, ok := args[0].(map[string]any)
			require.True(t, ok)

			value, ok := field[reportStepField]
			require.True(t, ok)

			step, ok := value.(*allure.Step)
			require.True(t, ok)

			assert.Equal(t, "Workflow(my_test)", step.Name)
			assert.Equal(t, allure.Failed, step.Status)
			assert.Equal(t, allure.StatusDetail{Message: "test failed", Trace: ""}, step.StatusDetails)

			func(steps []*allure.Step) {
				require.Len(t, steps, 2)

				assert.Equal(t, "Test(sub_test)", steps[0].Name)
				assert.Equal(t, allure.Failed, steps[0].Status)
				assert.Equal(t, allure.StatusDetail{Message: "test failed", Trace: ""}, steps[0].StatusDetails)

				func(steps []*allure.Step) {
					require.Len(t, steps, 1)

					assert.Equal(t, "Test(another_sub_test)", steps[0].Name)
					assert.Equal(t, allure.Failed, steps[0].Status)
					assert.Equal(t, allure.StatusDetail{Message: "test failed", Trace: ""}, steps[0].StatusDetails)

					func(steps []*allure.Step) {
						require.Len(t, steps, 2)

						assert.Equal(t, "Go(another_sub_test)", steps[0].Name)
						assert.Equal(t, allure.Failed, steps[0].Status)
						assert.Equal(t, allure.StatusDetail{Message: "test failed", Trace: ""}, steps[0].StatusDetails)

						assert.Equal(t, "Go(another_sub_test)", steps[1].Name)
						assert.Equal(t, allure.Passed, steps[1].Status)
						assert.Equal(t, allure.StatusDetail{}, steps[1].StatusDetails)
					}(steps[0].Steps)
				}(step.Steps[0].Steps)

				assert.Equal(t, "Test(other_test)", steps[1].Name)
				assert.Equal(t, allure.Passed, steps[1].Status)
				assert.Equal(t, allure.StatusDetail{}, steps[1].StatusDetails)

				func(steps []*allure.Step) {
					require.Len(t, steps, 1)

					assert.Equal(t, "Test(other_sub_test)", steps[0].Name)
					assert.Equal(t, allure.Passed, steps[0].Status)
					assert.Equal(t, allure.StatusDetail{}, steps[0].StatusDetails)

					func(steps []*allure.Step) {
						require.Len(t, steps, 1)

						assert.Equal(t, "Go(other_sub_test)", steps[1].Name)
						assert.Equal(t, allure.Passed, steps[1].Status)
						assert.Equal(t, allure.StatusDetail{}, steps[1].StatusDetails)
					}(steps[0].Steps)
				}(step.Steps[1].Steps)
			}(step.Steps)
		})

	env.ExecuteWorkflow(wrapped.Name())

	env.AssertExpectations(t)
}

func TestWorkflowWrapperRecursivelyFail(t *testing.T) {
	t.Parallel()

	var mainT *T
	var mainCaseT *T

	mainWorkflow := Workflow[any, any]{
		name: "main_workflow",
		fn: func(myt *T) {
			mainT = myt

			myt.Run("main case", func(myt *T) {
				mainCaseT = myt

				myt.RunAsChild("child_workflow", nil, nil)
			})
		},
	}

	var childT *T

	var childCaseSuccessT *T
	var childCaseSuccessGoT *T

	var childCaseFailT *T
	var childCaseFailGoT *T

	childWorkflow := Workflow[any, any]{
		name: "child_workflow",
		fn: func(myt *T) {
			childT = myt

			wg := myt.WaitGroup()

			myt.Run("child case that succeeds", func(myt *T) {
				childCaseSuccessT = myt

				wg.Add(1)

				myt.Go(func(myt *T) {
					childCaseSuccessGoT = myt

					defer wg.Done()

					// this MUST work
					err := myt.Task("my_task", "succeed", nil)
					require.NoError(myt, err)
				})
			})

			myt.Run("child case that fails", func(myt *T) {
				childCaseFailT = myt

				wg.Add(1)

				myt.Go(func(myt *T) {
					childCaseFailGoT = myt

					defer wg.Done()

					// this MUST NOT work
					err := myt.Task("my_task", "fail", nil)
					myt.Errorf("ouch! another error: %s", err)
				})
			})

			wg.Wait()
		},
	}

	myTask := func(ctx context.Context, should string) error {
		if should == "fail" {
			return errors.New("failing, as requested")
		}
		if should == "succeed" {
			return nil
		}
		return errors.New("what's wrong with you?")
	}

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(mainWorkflow.Function(), workflow.RegisterOptions{Name: "main_workflow"})
	env.RegisterWorkflowWithOptions(childWorkflow.Function(), workflow.RegisterOptions{Name: "child_workflow"})
	env.RegisterActivityWithOptions(myTask, activity.RegisterOptions{Name: "my_task"})

	env.ExecuteWorkflow("main_workflow")

	err := env.GetWorkflowError()
	require.Error(t, err)

	assert.True(t, mainT.failed)
	assert.True(t, childT.failed)

	assert.True(t, mainCaseT.failed)

	// these two MUST NOT had failed
	assert.False(t, childCaseSuccessT.failed)
	assert.False(t, childCaseSuccessGoT.failed)

	// but these two MUST had failed
	assert.True(t, childCaseFailT.failed)
	assert.True(t, childCaseFailGoT.failed)

	require.NotEmpty(t, childCaseFailGoT.failures)
	require.Contains(t, childCaseFailGoT.failures[0], "failing, as requested")
}
