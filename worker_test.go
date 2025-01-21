package tempo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func TestWorker(t *testing.T) {
	t.Parallel()

	w := &MockWorker{}

	w.On("RegisterWorkflowWithOptions", mock.Anything, mock.Anything).Times(4)
	w.On("RegisterActivity", mock.Anything).Times(1)

	Worker(w, Registry{
		Tests: []Test{
			NewTest(Example),
			NewTestWithInput(ExampleWithInput),
			NewTestWithOutput(ExampleWithOutput),
			NewTestWithInputAndOutput(ExampleWithInputAndOutput),
		},
		Tasks: []Task{
			ExampleActivity,
		},
	})

	w.AssertExpectations(t)
}

type MockWorker struct {
	mock.Mock
	worker.Worker
}

func (m *MockWorker) RegisterWorkflowWithOptions(w any, options workflow.RegisterOptions) {
	m.Called(w, options)
}

func (m *MockWorker) RegisterActivity(a any) {
	m.Called(a)
}

func ExampleActivity(ctx context.Context) error {
	return nil
}
