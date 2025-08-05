package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/workflow"
)

func TestWaitGroup(t *testing.T) {
	t.Parallel()

	var ctx workflow.Context

	mockedWaitGroup := &MockWaitGroup{}

	wg := WaitGroup{
		ctx:       ctx,
		WaitGroup: mockedWaitGroup,
	}

	wg.Wait()

	assert.Equal(t, 1, mockedWaitGroup.wait)
}
