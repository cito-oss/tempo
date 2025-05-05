package tempo

import (
	"errors"
	"strings"
)

var (
	ErrSetLimitWithGoroutines = errors.New("set limit with goroutines still active")
	ErrWorkflowExecute        = errors.New("execute workflow")
	ErrWorkflowAwait          = errors.New("await workflow")
	ErrWorkflow               = errors.New("workflow")
	ErrFuture                 = errors.New("future")
	ErrTest                   = errors.New("test")
)

const TestFailedErrorType = "TestFailedError"

type TestFailedError struct {
	msgs []string
}

func (e *TestFailedError) Error() string {
	if len(e.msgs) == 0 {
		return "test failed"
	}

	return strings.Join(e.msgs, "\n\n")
}

func NewTestFailedError(msgs []string) error {
	return &TestFailedError{msgs: msgs}
}
