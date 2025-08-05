package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannel(t *testing.T) {
	t.Parallel()

	t.Run("Send", func(t *testing.T) {
		t.Parallel()

		chann := &MockChannel{}

		c := &Channel{
			chann: chann,
		}

		c.Send(nil)

		assert.Equal(t, 1, chann.send)
	})

	t.Run("Send", func(t *testing.T) {
		t.Parallel()

		chann := &MockChannel{}

		c := &Channel{
			chann: chann,
		}

		c.Receive(nil)

		assert.Equal(t, 1, chann.receive)
	})

	t.Run("Close", func(t *testing.T) {
		t.Parallel()

		chann := &MockChannel{}

		c := &Channel{
			chann: chann,
		}

		c.Close()

		assert.Equal(t, 1, chann.close)
	})
}
