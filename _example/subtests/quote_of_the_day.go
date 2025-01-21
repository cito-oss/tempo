package subtests

import (
	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func QuoteOfTheDay(t *tempo.T) string {
	var quote string

	err := t.Task(tasks.RandomQuote, nil, &quote)
	require.NoError(t, err)

	assert.NotEmpty(t, quote)

	return quote
}
