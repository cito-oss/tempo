package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// stop() is reached during runtime.Goexit unwinding, on a goroutine invoke()
// spawned outside the dispatcher. If the workflow task ended in the meantime the
// dispatcher is no longer executing, and the wait inside stop() used to panic with
// "getState: illegal access from outside of workflow context", taking the worker
// process with it.
func TestStopAfterDispatcherStopped(t *testing.T) {
	t.Parallel()

	t.Run("nothing spawned", func(t *testing.T) {
		t.Parallel()

		escaped := escapedT(t, nil)

		assert.NotPanics(t, func() { escaped.stop() })
	})

	t.Run("Go spawned a goroutine", func(t *testing.T) {
		t.Parallel()

		escaped := escapedT(t, func(myt *T) { myt.Go(func(*T) {}) })

		assert.True(t, escaped.spawned)
		assert.NotPanics(t, func() { escaped.stop() })
	})
}

// cleanup() is reached on the same unwinding, one defer earlier than stop(). It
// builds a child T and runs bodies that talk to the workflow, so it panicked the
// same way — but only for a test that registered a cleanup, which is why the
// crashes in the wild landed in stop().
//
// Cleanups are attempted rather than skipped: a body doing plain Go work still
// completes, and only one that reaches for the workflow is abandoned. There is no
// way to ask the SDK whether a context is still live without touching it.
func TestCleanupAfterDispatcherStopped(t *testing.T) {
	t.Parallel()

	t.Run("a cleanup that runs a task is abandoned", func(t *testing.T) {
		t.Parallel()

		var escaped *T

		escaped = escapedT(t, func(myt *T) {
			myt.Cleanup(func() {
				_ = escaped.Task("myActivity", nil, nil)
			})
		})

		assert.NotPanics(t, func() { escaped.cleanup() })
	})

	t.Run("a cleanup that does plain work still runs", func(t *testing.T) {
		t.Parallel()

		var ran bool

		escaped := escapedT(t, func(myt *T) {
			myt.Cleanup(func() { ran = true })
		})

		assert.NotPanics(t, func() { escaped.cleanup() })

		// Only reachable because child no longer builds a WaitGroup: that call
		// used to panic before the first body was reached.
		assert.True(t, ran, "a cleanup touching nothing should still run")
	})

	t.Run("a real panic still surfaces", func(t *testing.T) {
		t.Parallel()

		env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

		// Inside a running workflow, so the dispatcher is live and the panic is
		// the cleanup genuinely failing rather than a dead context. Only the SDK's
		// dispatcher panic is swallowed; anything else must not be hidden.
		env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
			myt := &T{ctx: ctx, name: "boom", cleanups: []func(){
				func() { panic("something else") },
			}}

			assert.PanicsWithValue(t, "something else", func() { myt.cleanup() })

			return nil
		}, workflow.RegisterOptions{Name: "myWorkflow"})

		env.ExecuteWorkflow("myWorkflow")
	})
}

// escapedT runs a workflow to completion and hands back the T it built, so the
// caller holds a workflow context whose dispatcher has stopped executing.
func escapedT(t *testing.T, during func(*T)) *T {
	t.Helper()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()

	var escaped *T

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		escaped = &T{ctx: ctx, wg: workflow.NewWaitGroup(ctx)}

		if during != nil {
			during(escaped)
		}

		return nil
	}, workflow.RegisterOptions{Name: "myWorkflow"})

	env.ExecuteWorkflow("myWorkflow")

	return escaped
}
