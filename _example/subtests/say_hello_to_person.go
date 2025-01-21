package subtests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Person struct {
	Name      string
	Age       int
	Greetings string
}

func SayHelloToPerson(t *tempo.T, name string) *Person {
	wg := t.WaitGroup()

	var greetings string
	var age int

	t.Run("greet person", func(t *tempo.T) {
		wg.Add(1)

		t.Go(func(t *tempo.T) {
			defer wg.Done()

			err := t.Task(tasks.Greeter, name, &greetings)
			require.NoError(t, err)

			assert.Equal(t, "Hello "+name, greetings)
		})
	})

	t.Run("get person age", func(t *tempo.T) {
		wg.Add(1)

		t.Go(func(t *tempo.T) {
			defer wg.Done()

			err := t.Task(tasks.PersonAge, name, &age)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, age, 1)
		})
	})

	wg.Wait()

	return &Person{
		Name:      name,
		Age:       age,
		Greetings: greetings,
	}
}
