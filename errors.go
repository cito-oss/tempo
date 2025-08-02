package tempo

import (
	"errors"
	"strings"
)

var (
	ErrSetLimitWithGoroutines = errors.New("set limit with goroutines")
	ErrWorkflowExecute        = errors.New("workflow execute")
	ErrWorkflowResult         = errors.New("workflow result")
	ErrTaskExecute            = errors.New("task execute")
	ErrTaskResult             = errors.New("task result")
	ErrPlanRun                = errors.New("plan run")
)

const TestFailedErrorType = "TestFailedError"

type TestFailedError struct {
	msgs []string
}

func (e TestFailedError) Error() string {
	if len(e.msgs) == 0 {
		return "test failed"
	}

	return strings.Join(e.msgs, "\n\n")
}

func NewTestFailedError(msgs []string) error {
	return TestFailedError{msgs: msgs}
}
