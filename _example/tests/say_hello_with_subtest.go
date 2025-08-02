package tests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SayHelloWithSubtest(t *tempo.T) {
	t.Run("say hello to John Doe", func(t *tempo.T) {
		person := &Person{}

		t.RunAsChild(SayHelloToPerson, "John Doe", person)

		assert.Equal(t, "John Doe", person.Name)
		assert.Equal(t, 35, person.Age)
		assert.Equal(t, "Hello John Doe", person.Greetings)
	})

	t.Run("say hello to Jane Doe", func(t *tempo.T) {
		person := &Person{}

		t.RunAsChild(SayHelloToPerson, "Jane Doe", person)

		assert.Equal(t, "Jane Doe", person.Name)
		assert.Equal(t, 27, person.Age)
		assert.Equal(t, "Hello Jane Doe", person.Greetings)
	})

	t.Run("say hello to World", func(t *tempo.T) {
		person := &Person{}

		t.RunAsChild(SayHelloToPerson, "World", person)

		assert.Equal(t, "World", person.Name)
		assert.Equal(t, int(4.54e9), person.Age)
		assert.Equal(t, "Hello World", person.Greetings)
	})

	t.Run("quote of the day", func(t *tempo.T) {
		var quote string

		t.RunAsChild(QuoteOfTheDay, nil, &quote)

		assert.NotEmpty(t, quote)
	})

	t.Run("say hello to Guina", func(t *tempo.T) {
		// this is an example of a failing test
		// this person doesn't exists therefore the task PersonAge will fail
		person := &Person{}

		t.RunAsChild(SayHelloToPerson, "Guina", person)

		assert.Equal(t, "Guina", person.Name)
	})
}

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

func QuoteOfTheDay(t *tempo.T) string {
	var quote string

	err := t.Task(tasks.RandomQuote, nil, &quote)
	require.NoError(t, err)

	assert.NotEmpty(t, quote)

	return quote
}
