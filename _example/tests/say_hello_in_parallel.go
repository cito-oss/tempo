package tests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SayHelloInParallel(t *tempo.T) {
	names := []string{"John Doe", "Jane Doe", "World"}

	wg := t.WaitGroup()

	for _, name := range names {
		wg.Add(1)

		t.Go(func(t *tempo.T) {
			defer wg.Done()

			var greetings string

			err := t.Task(tasks.Greeter, name, &greetings)
			require.NoError(t, err)

			assert.Equal(t, "Hello "+name, greetings)
		})
	}

	wg.Wait()
}
