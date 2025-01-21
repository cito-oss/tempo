package tests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SayHelloWorld(t *tempo.T) {
	var greetings string

	err := t.Task(tasks.Greeter, nil, &greetings)
	require.NoError(t, err)

	assert.Equal(t, "Hello World", greetings)
}
