package tests

import (
	"time"

	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func UnknownPerson(t *tempo.T) {
	t.SetActivityOptions(workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
			NonRetryableErrorTypes: []string{
				tasks.UnknownPersonErrorType, // do not retry when person is unknown
			},
		},
	})

	var greetings string

	err := t.Task(tasks.PersonAge, "Guina", &greetings)
	require.Error(t, err)

	assert.Empty(t, greetings)
}
