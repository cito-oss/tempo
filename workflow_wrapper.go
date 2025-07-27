package tempo

import (
	"go.temporal.io/sdk/workflow"
)

type workflowWrapper[INPUT any, OUTPUT any] struct {
	fn             func(*T)
	fnWithIn       func(*T, INPUT)
	fnWithOut      func(*T) OUTPUT
	fnWithInAndOut func(*T, INPUT) OUTPUT
	name           string
}

// Name returns the name of the Workflow
func (c *workflowWrapper[I, O]) Name() string {
	return c.name
}

// Function returns the function that should be given to Temporal
func (c *workflowWrapper[I, O]) Function() any {
	return c.workflow
}

// workflow is the function that is given to temporal SDK.
// The `input` must use generics in order to temporal SDK infer the type
func (c *workflowWrapper[I, O]) workflow(ctx workflow.Context, input I) (O, error) {
	logger := workflow.GetLogger(ctx)

	exitChan := workflow.NewChannel(ctx)
	errsChan := workflow.NewChannel(ctx)

	var done bool
	var output O

	workflow.Go(ctx, func(ctx workflow.Context) {
		t := &T{
			name:   c.name,
			ctx:    ctx,
			logger: logger,
			exit:   exitChan,
			errs:   errsChan,
		}

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

		done = true

		exitChan.Close()
		errsChan.Close()
	})

	selector := workflow.NewSelector(ctx)

	var exit bool
	var errs []string

	selector.AddReceive(exitChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &exit)
	})

	selector.AddReceive(errsChan, func(c workflow.ReceiveChannel, more bool) {
		var msg string
		c.Receive(ctx, &msg)

		errs = append(errs, msg)
	})

	for {
		selector.Select(ctx)

		if exit || done {
			break
		}
	}

	var err error

	if exit {
		err = TestExittedError{}
	}

	if len(errs) > 0 {
		err = NewTestFailedError(errs)
	}

	if err != nil {
		var zero O
		return zero, err
	}

	return output, nil
}
