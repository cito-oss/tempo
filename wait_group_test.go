package tempo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/workflow"
)

func TestWaitGroup(t *testing.T) {
	t.Parallel()

	ctx := &MockWorkflowContext{}

	mockedWaitGroup := &MockWorkflowWaitGroup{}

	wg := WaitGroup{
		ctx:       ctx,
		WaitGroup: mockedWaitGroup,
	}

	wg.Wait()

	assert.Equal(t, ctx, mockedWaitGroup.calls[0])
}

type MockWorkflowContext struct {
	workflow.Context
}

type MockWorkflowWaitGroup struct {
	workflow.WaitGroup
	calls []workflow.Context
}

func (m *MockWorkflowWaitGroup) Wait(ctx workflow.Context) {
	m.calls = append(m.calls, ctx)
}
