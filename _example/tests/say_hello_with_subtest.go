package tests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/subtests"
	"github.com/stretchr/testify/assert"
)

func SayHelloWithSubtest(t *tempo.T) {
	t.Run("say hello to John Doe", func(t *tempo.T) {
		person := &subtests.Person{}

		t.RunAsChild(subtests.SayHelloToPerson, "John Doe", person)

		assert.Equal(t, "John Doe", person.Name)
		assert.Equal(t, 35, person.Age)
		assert.Equal(t, "Hello John Doe", person.Greetings)
	})

	t.Run("say hello to Jane Doe", func(t *tempo.T) {
		person := &subtests.Person{}

		t.RunAsChild(subtests.SayHelloToPerson, "Jane Doe", person)

		assert.Equal(t, "Jane Doe", person.Name)
		assert.Equal(t, 27, person.Age)
		assert.Equal(t, "Hello Jane Doe", person.Greetings)
	})

	t.Run("say hello to World", func(t *tempo.T) {
		person := &subtests.Person{}

		t.RunAsChild(subtests.SayHelloToPerson, "World", person)

		assert.Equal(t, "World", person.Name)
		assert.Equal(t, int(4540000000), person.Age) // 4.54e9
		assert.Equal(t, "Hello World", person.Greetings)
	})

	t.Run("quote of the day", func(t *tempo.T) {
		var quote string

		t.RunAsChild(subtests.QuoteOfTheDay, nil, &quote)

		assert.NotEmpty(t, quote)
	})
}
