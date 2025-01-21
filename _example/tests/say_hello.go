package tests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SayHello(t *tempo.T, name string) {
	var greetings string

	err := t.Task(tasks.Greeter, name, &greetings)
	require.NoError(t, err)

	assert.Equal(t, "Hello "+name, greetings)
}
