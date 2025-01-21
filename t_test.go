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
			name: "my_test",
			ctx:  ctx,
		}

		myt.Go(func(myt *T) {
			called = true
			assert.Equal(t, "my_test", myt.name)
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

	t.Run("fail workflow await", func(t *testing.T) {
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

			assert.ErrorIs(t, err, ErrWorkflowAwait)
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

			assert.ErrorIs(t, err, ErrFuture)
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

	t.Run("fail workflow await", func(t *testing.T) {
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

			assert.ErrorIs(t, err, ErrWorkflowAwait)

			assert.Equal(t, "", given)

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

			assert.ErrorIs(t, err, ErrFuture)

			assert.Equal(t, "", given)

			return "", errors.New("activity failed")
		}

		myWorkflow := func(ctx workflow.Context) error {
			myt := &T{ctx: ctx}

			var given string

			myt.RunAsChild("myChildWorkflow", "world", &given)

			assert.Equal(t, "", given)

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

			assert.ErrorIs(t, err, ErrFuture)

			assert.Equal(t, "", given)

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

			assert.Equal(t, "", given)

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

	var called bool

	myTestFn := func(ctx workflow.Context) error {
		called = true

		myt := &T{ctx: ctx}

		wg := myt.WaitGroup()
		assert.Equal(t, ctx, wg.ctx)

		return nil
	}

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

	env.RegisterWorkflow(myTestFn)
	env.ExecuteWorkflow(myTestFn)

	assert.True(t, called)
}

func TestTRun(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		logger := &MockLogger{}

		myt := &T{logger: logger}

		var called bool

		myt.Run("my_test", func(myt *T) {
			called = true
		})

		require.NotEmpty(t, logger.calls)
		assert.Equal(t, "run test: [name my_test]", logger.calls[0])

		assert.True(t, called)
	})

	t.Run("fail now", func(t *testing.T) {
		t.Parallel()

		logger := &MockLogger{}
		exit := &MockChannel{}

		myt := &T{
			logger: logger,
			exit:   exit,
		}

		var called bool

		myt.Run("my_test", func(myt *T) {
			called = true
			myt.FailNow()
		})

		require.Len(t, logger.calls, 2)
		assert.Equal(t, "run test: [name my_test]", logger.calls[0])
		assert.Equal(t, "test failed: [name my_test]", logger.calls[1])

		require.NotEmpty(t, exit.calls)
		assert.Equal(t, "true", exit.calls[0])

		assert.True(t, called)
	})

	t.Run("errorf", func(t *testing.T) {
		t.Parallel()

		logger := &MockLogger{}
		msgs := &MockChannel{}

		myt := &T{
			logger: logger,
			msgs:   msgs,
		}

		var called bool

		myt.Run("my_test", func(myt *T) {
			called = true
			myt.Errorf("ohh no! %s: %s", "what happened?", "nothing!")
		})

		require.Len(t, logger.calls, 2)
		assert.Equal(t, "run test: [name my_test]", logger.calls[0])
		assert.Equal(t, "ohh no! what happened?: nothing!: [name my_test]", logger.calls[1])

		require.NotEmpty(t, msgs.calls)
		assert.Equal(t, "ohh no! what happened?: nothing!", msgs.calls[0])

		assert.True(t, called)
	})

	t.Run("warnf", func(t *testing.T) {
		t.Parallel()

		logger := &MockLogger{}
		msgs := &MockChannel{}

		myt := &T{
			logger: logger,
			msgs:   msgs,
		}

		var called bool

		myt.Run("my_test", func(myt *T) {
			called = true
			myt.Warnf("check this out")
		})

		require.Len(t, logger.calls, 2)
		assert.Equal(t, "run test: [name my_test]", logger.calls[0])
		assert.Equal(t, "check this out: [name my_test]", logger.calls[1])

		assert.True(t, called)
	})
}

// MockLogger is a logger that discards all log messages.
type MockLogger struct {
	log.Logger
	calls []string
}

func (m *MockLogger) Info(msg string, keyvals ...interface{}) {
	m.calls = append(m.calls, fmt.Sprintf("%s: %+v", msg, keyvals))
}

func (m *MockLogger) Warn(msg string, keyvals ...interface{}) {
	m.calls = append(m.calls, fmt.Sprintf("%s: %+v", msg, keyvals))
}

func (m *MockLogger) Error(msg string, keyvals ...interface{}) {
	m.calls = append(m.calls, fmt.Sprintf("%s: %+v", msg, keyvals))
}

type MockChannel struct {
	workflow.Channel
	calls []string
}

func (m *MockChannel) Send(ctx workflow.Context, value any) {
	m.calls = append(m.calls, fmt.Sprintf("%+v", value))
}
