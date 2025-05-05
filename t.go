package tempo

import (
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var DefaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		MaximumAttempts: 1,
	},
}

type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

var _ TestingT = &T{}

type T struct {
	ctx     workflow.Context
	logger  log.Logger
	exit    workflow.Channel
	errs    workflow.Channel
	options *workflow.ActivityOptions
	parent  *T
	name    string
	failed  bool
}

func (t *T) fail() {
	t.failed = true
	if t.parent != nil {
		t.parent.fail()
	}
}

func (t *T) Errorf(format string, args ...any) {
	t.fail()
	msg := fmt.Sprintf(format, args...)
	t.logger.Error(msg, "name", t.name)
	t.errs.Send(t.ctx, msg)
}

func (t *T) Warnf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	t.logger.Warn(msg, "name", t.name)
}

func (t *T) FailNow() {
	t.fail()
	t.logger.Error("test failed", "name", t.name)
	t.exit.Send(t.ctx, true)
}

func (t *T) SetActivityOptions(options workflow.ActivityOptions) {
	t.options = &options
}

func (t *T) Run(name string, fn func(*T)) {
	t.logger.Info("run test", "name", name)
	fn(&T{
		name:    name,
		ctx:     t.ctx,
		logger:  t.logger,
		options: t.options,
		exit:    t.exit,
		errs:    t.errs,
		parent:  t,
	})
}

func (t *T) WaitGroup() *WaitGroup {
	return &WaitGroup{
		WaitGroup: workflow.NewWaitGroup(t.ctx),
		ctx:       t.ctx,
	}
}

func (t *T) Go(fn func(t *T)) {
	workflow.Go(t.ctx, func(ctx workflow.Context) {
		fn(&T{
			ctx:     ctx,
			name:    t.name,
			logger:  t.logger,
			options: t.options,
			exit:    t.exit,
			errs:    t.errs,
		})
	})
}

func (t *T) RunAsChild(fn any, input any, output any) {
	name := getFunctionName(fn)

	err := t.invoke(t.ctx, name, input, output)
	if err != nil {
		t.Errorf("run as child: %s", err)
		t.FailNow()
	}
}

func (t *T) Task(task any, input any, output any) error {
	opts := DefaultActivityOptions

	if t.options != nil {
		opts = *t.options
	}

	// set Summary with test name
	opts.Summary = t.name

	ctx := workflow.WithActivityOptions(t.ctx, opts)

	future := workflow.ExecuteActivity(ctx, task, input)

	err := workflow.Await(t.ctx, future.IsReady)
	if err != nil {
		return errors.Join(ErrWorkflowAwait, err)
	}

	err = future.Get(t.ctx, output)
	if err != nil {
		return errors.Join(ErrFuture, err)
	}

	return nil
}

func (t *T) invoke(ctx workflow.Context, name string, input any, output any) error {
	info := workflow.GetInfo(ctx)

	var retryPolicy *temporal.RetryPolicy
	if t.options != nil {
		retryPolicy = t.options.RetryPolicy
	}

	// TODO expose this to outside control
	opts := workflow.ChildWorkflowOptions{
		TaskQueue:     info.TaskQueueName,
		RetryPolicy:   retryPolicy,
		StaticSummary: t.name,
	}

	ctx = workflow.WithChildOptions(ctx, opts)

	future := workflow.ExecuteChildWorkflow(ctx, name, input)

	err := workflow.Await(ctx, future.IsReady)
	if err != nil {
		return errors.Join(ErrWorkflowAwait, err)
	}

	err = future.Get(ctx, output)
	if err != nil {
		return errors.Join(ErrFuture, err)
	}

	return nil
}
