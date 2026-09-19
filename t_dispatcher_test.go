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
