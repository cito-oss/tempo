package tests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SayHelloToEveryone(t *tempo.T) {
	t.Run("say hello to John Doe", func(t *tempo.T) {
		var greetings string

		err := t.Task(tasks.Greeter, "John Doe", &greetings)
		require.NoError(t, err)

		assert.Equal(t, "Hello John Doe", greetings)
	})

	t.Run("say hello to Jane Doe", func(t *tempo.T) {
		var greetings string

		err := t.Task(tasks.Greeter, "Jane Doe", &greetings)
		require.NoError(t, err)

		assert.Equal(t, "Hello Jane Doe", greetings)
	})

	t.Run("say hello to World", func(t *tempo.T) {
		var greetings string

		err := t.Task(tasks.Greeter, "World", &greetings)
		require.NoError(t, err)

		assert.Equal(t, "Hello World", greetings)
	})
}
