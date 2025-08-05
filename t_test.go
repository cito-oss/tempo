package tempo

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestTGo(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

	var called bool

	myWorkflow := func(ctx workflow.Context) error {
		myt := &T{
			ctx: ctx,
			wg:  workflow.NewWaitGroup(ctx),
		}

		myt.Go(func(myt *T) {
			called = true
		})

		return nil
	}

	env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})

	env.ExecuteWorkflow("myWorkflow")

	assert.True(t, called)
}

func TestTActivity(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			myt.SetActivityOptions(DefaultActivityOptions)

			var given string

			err := myt.Task("myActivity", "world", &given)
			require.NoError(t, err)

			assert.Equal(t, "hello world", given)

			return nil
		}

		env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})
		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("myWorkflow")

		assert.True(t, called)
	})

	t.Run("fail after cancel workflow", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			env.CancelWorkflow()

			var given string

			err := myt.Task("myActivity", "world", &given)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrTaskExecute)
			assert.Empty(t, given)

			return nil
		}

		env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})
		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("myWorkflow")

		assert.False(t, called)
	})

	t.Run("fail activity", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "", errors.New("something went wrong")
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			var given string

			err := myt.Task("myActivity", "world", &given)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrTaskResult)
			assert.Empty(t, given)

			return nil
		}

		env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})
		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("myWorkflow")

		assert.True(t, called)
	})
}

func TestTRunAsChild(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called bool

		myActivity := func(ctx context.Context, name string) (string, error) {
			called = true
			return "hello " + name, nil
		}

		myChildWorkflow := func(ctx workflow.Context, name string) (string, error) {
			require.Equal(t, "world", name)

			myt := &T{ctx: ctx}

			var given string

			err := myt.Task("myActivity", name, &given)
			require.NoError(t, err)

			assert.Equal(t, "hello world", given)

			return given, nil
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			var given string

			myt.RunAsChild("myChildWorkflow", "world", &given)

			assert.Equal(t, "hello world", given)

			return nil
		}

		env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})
		env.RegisterWorkflowWithOptions(myChildWorkflow, workflow.RegisterOptions{Name: "myChildWorkflow"})

		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("myWorkflow")

		require.NoError(t, env.GetWorkflowError())

		assert.True(t, called)
	})

	t.Run("fail after cancel workflow", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called atomic.Bool // to avoid race condition, use atomic

		myActivity := func(ctx context.Context, name string) (string, error) {
			called.Store(true)
			return "hello " + name, nil
		}

		myChildWorkflow := func(ctx workflow.Context, name string) (string, error) {
			require.Equal(t, "world", name)

			myt := &T{ctx: ctx}

			var given string

			env.CancelWorkflow()

			err := myt.Task("myActivity", name, &given)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrTaskExecute)
			assert.Empty(t, given)

			return "", nil
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			var given string

			myt.RunAsChild("myChildWorkflow", "world", &given)

			return nil
		}

		env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})
		env.RegisterWorkflowWithOptions(myChildWorkflow, workflow.RegisterOptions{Name: "myChildWorkflow"})

		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("myWorkflow")

		require.Error(t, env.GetWorkflowError())

		assert.False(t, called.Load())
	})

	t.Run("fail activity", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called atomic.Bool // to avoid race condition, use atomic

		myActivity := func(ctx context.Context, name string) (string, error) {
			called.Store(true)
			return "", errors.New("something went wrong")
		}

		myChildWorkflow := func(ctx workflow.Context, name string) (string, error) {
			require.Equal(t, "world", name)

			myt := &T{ctx: ctx}

			var given string

			err := myt.Task("myActivity", name, &given)
			require.Error(t, err)
			require.ErrorContains(t, err, "something went wrong")
			assert.Empty(t, given)

			return "", errors.New("activity failed")
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			var given string

			myt.RunAsChild("myChildWorkflow", "world", &given)

			assert.Empty(t, given)

			return nil
		}

		env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})
		env.RegisterWorkflowWithOptions(myChildWorkflow, workflow.RegisterOptions{Name: "myChildWorkflow"})

		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("myWorkflow")

		require.Error(t, env.GetWorkflowError())

		assert.True(t, called.Load())
	})

	t.Run("with retry policy", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		var called atomic.Int32 // to avoid race condition, use atomic

		myActivity := func(ctx context.Context, name string) (string, error) {
			called.Add(1)
			return "", errors.New("something went wrong")
		}

		myChildWorkflow := func(ctx workflow.Context, name string) (string, error) {
			require.Equal(t, "world", name)

			myt := &T{ctx: ctx}

			var given string

			err := myt.Task("myActivity", name, &given)
			require.Error(t, err)
			require.ErrorContains(t, err, "something went wrong")
			assert.Empty(t, given)

			return "", errors.New("activity failed")
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			myt.SetActivityOptions(workflow.ActivityOptions{
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Millisecond,
					BackoffCoefficient: 2,
					MaximumAttempts:    3,
				},
			})

			var given string

			myt.RunAsChild("myChildWorkflow", "world", &given)

			assert.Empty(t, given)

			return nil
		}

		env.RegisterWorkflowWithOptions(myWorkflow, workflow.RegisterOptions{Name: "myWorkflow"})
		env.RegisterWorkflowWithOptions(myChildWorkflow, workflow.RegisterOptions{Name: "myChildWorkflow"})

		env.RegisterActivityWithOptions(myActivity, activity.RegisterOptions{Name: "myActivity"})

		env.ExecuteWorkflow("myWorkflow")

		require.Error(t, env.GetWorkflowError())

		assert.Equal(t, int32(3), called.Load())
	})
}

func TestTWaitGroup(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

	var called bool

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		called = true

		myt := &T{ctx: ctx}

		wg := myt.WaitGroup()
		assert.Equal(t, ctx, wg.ctx)

		return nil
	})

	assert.True(t, called)
}

func TestTBufferedChannel(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

	var called bool

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		called = true

		myt := &T{ctx: ctx}

		chann := myt.BufferedChannel(3)
		assert.Equal(t, ctx, chann.ctx)

		return nil
	})

	assert.True(t, called)
}

func TestTRun(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()

		logger := &MockLogger{}

		var called bool

		env.ExecuteWorkflow(func(ctx workflow.Context) error {
			myt := &T{
				ctx:    ctx,
				logger: logger,
			}

			myt.Run("my_test", func(myt *T) {
				called = true
			})

			wg := myt.WaitGroup()
			assert.Equal(t, ctx, wg.ctx)

			return nil
		})

		require.NotEmpty(t, logger.info)

		assert.Equal(t, "run test: [name my_test]", logger.info[0])

		assert.True(t, called)
	})

	t.Run("fail now", func(t *testing.T) {
		t.Parallel()

		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()

		logger := &MockLogger{}

		called := false
		uncalled := true

		var parentt *T
		var childt *T

		env.ExecuteWorkflow(func(ctx workflow.Context) error {
			parentt = &T{
				ctx:    ctx,
				logger: logger,
			}

			parentt.Run("my_test", func(myt *T) {
				childt = myt

				called = true

				myt.FailNow()

				uncalled = false
			})

			return nil
		})

		require.Len(t, logger.info, 1)

		assert.Equal(t, "run test: [name my_test]", logger.info[0])

		assert.True(t, called)
		assert.True(t, uncalled)

		require.NotNil(t, childt)
		require.NotNil(t, parentt)

		assert.True(t, childt.failed)
		assert.True(t, parentt.failed)

		assert.Empty(t, childt.failures)
		assert.Empty(t, parentt.failures)
	})

	t.Run("errorf", func(t *testing.T) {
		t.Parallel()

		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()

		logger := &MockLogger{}

		var before bool
		var after bool

		var parentt *T
		var childt *T

		env.ExecuteWorkflow(func(ctx workflow.Context) error {
			parentt = &T{
				ctx:    ctx,
				logger: logger,
			}

			parentt.Run("my_test", func(myt *T) {
				childt = myt

				before = true

				myt.Errorf("ohh no! %s: %s", "what happened?", "nothing!")

				after = true
			})

			return nil
		})

		require.Len(t, logger.info, 1)
		require.Len(t, logger.error, 1)

		assert.Equal(t, "run test: [name my_test]", logger.info[0])
		assert.Equal(t, "ohh no! what happened?: nothing!: [name my_test]", logger.error[0])

		assert.True(t, before)
		assert.True(t, after)

		require.NotNil(t, childt)
		require.NotNil(t, parentt)

		assert.True(t, childt.failed)
		assert.True(t, parentt.failed)

		require.Len(t, childt.failures, 1)
		assert.Equal(t, "ohh no! what happened?: nothing!", childt.failures[0])

		require.Empty(t, parentt.failures)
	})

	t.Run("warnf", func(t *testing.T) {
		t.Parallel()

		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()

		logger := &MockLogger{}

		var called bool

		env.ExecuteWorkflow(func(ctx workflow.Context) error {
			myt := &T{
				ctx:    ctx,
				logger: logger,
			}

			myt.Run("my_test", func(myt *T) {
				called = true
				myt.Warnf("check this out")
			})

			return nil
		})

		require.Len(t, logger.info, 1)
		require.Len(t, logger.warn, 1)

		assert.Equal(t, "run test: [name my_test]", logger.info[0])
		assert.Equal(t, "check this out: [name my_test]", logger.warn[0])

		assert.True(t, called)
	})
}

func TestT(t *testing.T) {
	t.Parallel()

	t.Run("stop with no steps", func(t *testing.T) {
		t.Parallel()

		mockWaitGroup := &MockWaitGroup{}

		myt := T{
			wg: mockWaitGroup,
		}

		myt.stop()

		assert.Equal(t, 1, mockWaitGroup.wait)
	})
}

// MockLogger is a logger that discards all log messages.
type MockLogger struct {
	log.Logger
	info  []string
	warn  []string
	error []string
}

func (m *MockLogger) Info(msg string, keyvals ...any) {
	m.info = append(m.info, fmt.Sprintf("%s: %+v", msg, keyvals))
}

func (m *MockLogger) Warn(msg string, keyvals ...any) {
	m.warn = append(m.warn, fmt.Sprintf("%s: %+v", msg, keyvals))
}

func (m *MockLogger) Error(msg string, keyvals ...any) {
	m.error = append(m.error, fmt.Sprintf("%s: %+v", msg, keyvals))
}

type MockChannel struct {
	workflow.Channel
	send    int
	receive int
	close   int
}

func (m *MockChannel) Send(ctx workflow.Context, value any) {
	m.send++
}

func (m *MockChannel) Receive(ctx workflow.Context, valuePtr any) (more bool) {
	m.receive++
	return false
}

func (m *MockChannel) Close() {
	m.close++
}

type MockWaitGroup struct {
	workflow.WaitGroup
	wait int
}

func (m *MockWaitGroup) Wait(ctx workflow.Context) {
	m.wait++
}
