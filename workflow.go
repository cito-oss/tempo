package tempo

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/ozontech/allure-go/pkg/allure"
	"go.temporal.io/sdk/workflow"
)

// Workflow is a wrapper for Temporal Workflow
type Workflow[INPUT any, OUTPUT any] struct {
	fn             func(*T)
	fnWithIn       func(*T, INPUT)
	fnWithOut      func(*T) OUTPUT
	fnWithInAndOut func(*T, INPUT) OUTPUT
	name           string
}

// Name returns the name of the Workflow
func (c *Workflow[I, O]) Name() string {
	return c.name
}

// Function returns the function that should be given to Temporal
func (c *Workflow[I, O]) Function() any {
	return c.workflow
}

// workflow is the function that is given to temporal SDK.
// The `input` must use generics in order to temporal SDK infer the type
func (c *Workflow[I, O]) workflow(ctx workflow.Context, input I) (O, error) {
	logger := workflow.GetLogger(ctx)

	step := &allure.Step{
		Name:  fmt.Sprintf("Workflow(%s)", c.name),
		Start: workflow.Now(ctx).UnixMilli(),
	}

	if param, ok := newParameter("input", input); ok {
		step.Parameters = append(step.Parameters, param)
	}

	t := &T{
		name:   c.name,
		ctx:    ctx,
		logger: logger,
		step:   step,
		wg:     workflow.NewWaitGroup(ctx),
	}

	var output O

	invoke(func() {
		switch {
		case c.fn != nil:
			c.fn(t)

		case c.fnWithIn != nil:
			c.fnWithIn(t, input)

		case c.fnWithOut != nil:
			output = c.fnWithOut(t)

		case c.fnWithInAndOut != nil:
			output = c.fnWithInAndOut(t, input)
		}
	})

	step.Stop = workflow.Now(ctx).UnixMilli()
	step.Status = allure.Passed

	if t.failed {
		step.Status = allure.Failed
		step.StatusDetails = allure.StatusDetail{
			Message: "test failed",
			Trace:   strings.Join(t.failures, "\n"),
		}
	}

	if param, ok := newParameter("output", output); ok {
		step.Parameters = append(step.Parameters, param)
	}

	memo := map[string]any{
		reportStepField: step,
	}

	if err := workflow.UpsertMemo(ctx, memo); err != nil {
		logger.Warn("report: failed to upsert memo", "error", err)
	}

	if t.failed {
		var zero O
		return zero, NewTestFailedError(t.failures)
	}

	return output, nil
}

// invoke calls the given function but in a goroutine, so it can be aborted using runtime.Goexit
func invoke(fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
	wg.Wait()
}

// newParameter uses reflection to parse given name/value into an allure newParameter
func newParameter(name string, data any) (*allure.Parameter, bool) {
	vof := reflect.ValueOf(data)

	if !vof.IsValid() {
		return nil, false
	}

	if vof.IsZero() {
		return nil, false
	}

	tmp, err := json.Marshal(data)
	if err != nil {
		return nil, false
	}

	var value any

	err = json.Unmarshal(tmp, &value)
	if err != nil {
		return nil, false
	}

	return &allure.Parameter{
		Name:  name,
		Value: value,
	}, true
}
