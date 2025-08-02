package tempo

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/ozontech/allure-go/pkg/allure"
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
	Errorf(format string, args ...any)
	FailNow()
}

var _ TestingT = &T{}

type T struct {
	ctx      workflow.Context
	logger   log.Logger
	options  *workflow.ActivityOptions
	parent   *T
	name     string
	failures []string
	failed   bool
	step     *allure.Step
	wg       workflow.WaitGroup
}

func (t *T) child(name string) *T {
	return &T{
		name:    name,
		ctx:     t.ctx,
		logger:  t.logger,
		options: t.options,
		parent:  t,
		wg:      workflow.NewWaitGroup(t.ctx),
	}
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
	t.failures = append(t.failures, msg)
}

func (t *T) Warnf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	t.logger.Warn(msg, "name", t.name)
}

func (t *T) FailNow() {
	t.fail()
	runtime.Goexit()
}

func (t *T) SetActivityOptions(options workflow.ActivityOptions) {
	t.options = &options
}

func (t *T) start(name string) {
	t.step = &allure.Step{
		Name:  name,
		Start: workflow.Now(t.ctx).UnixMilli(),
	}

	if t.parent != nil && t.parent.step != nil {
		t.parent.step.Steps = append(t.parent.step.Steps, t.step)
	}
}

func (t *T) stop() {
	t.wg.Wait(t.ctx)

	if t.step == nil {
		return
	}

	t.step.Stop = workflow.Now(t.ctx).UnixMilli()

	t.step.Status = allure.Passed

	if t.failed {
		t.step.Status = allure.Failed
		t.step.StatusDetails = allure.StatusDetail{
			Message: "test failed",
			Trace:   strings.Join(t.failures, "\n"),
		}
	}
}

func (t *T) Run(name string, fn func(*T)) {
	t.logger.Info("run test", "name", name)

	child := t.child(name)

	child.start(fmt.Sprintf("Test(%s)", name))
	defer child.stop()

	invoke(func() { fn(child) })
}

func (t *T) WaitGroup() *WaitGroup {
	return &WaitGroup{
		WaitGroup: workflow.NewWaitGroup(t.ctx),
		ctx:       t.ctx,
	}
}

func (t *T) Go(fn func(t *T)) {
	t.wg.Add(1)

	workflow.Go(t.ctx, func(ctx workflow.Context) {
		child := t.child(t.name) // create a child with same name...

		child.ctx = ctx // ...but replace the context

		child.start(fmt.Sprintf("Go(%s)", t.name))
		defer child.stop()
		defer t.wg.Done()

		invoke(func() { fn(child) })
	})
}

func (t *T) RunAsChild(fn any, input any, output any) {
	name := getFunctionName(fn)

	step := &allure.Step{
		Name:  fmt.Sprintf("ChildWorkflow(%s)", name),
		Start: workflow.Now(t.ctx).UnixMilli(),
	}

	if t.step != nil {
		t.step.Steps = append(t.step.Steps, step)
	}

	info := workflow.GetInfo(t.ctx)

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

	ctx := workflow.WithChildOptions(t.ctx, opts)

	future := workflow.ExecuteChildWorkflow(ctx, name, input)

	var err error

	defer func() {
		step.Stop = workflow.Now(t.ctx).UnixMilli()
		step.Status = allure.Passed

		if err != nil {
			step.Status = allure.Failed
			step.StatusDetails = allure.StatusDetail{
				Message: "child worlfkow failed",
				Trace:   err.Error(),
			}
		}

		var childexec workflow.Execution

		err := future.GetChildWorkflowExecution().Get(ctx, &childexec)
		if err != nil {
			t.logger.Warn("report: failed to get child workflow execution",
				"error", err,
				"name", name,
			)
			return
		}

		step.Attachments = append(step.Attachments, &allure.Attachment{
			Name:   reportReplaceFlag,
			Source: reportReplaceID(childexec.ID, childexec.RunID),
		})
	}()

	err = workflow.Await(ctx, future.IsReady)
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	err = future.Get(ctx, output)
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}
}

func (t *T) Task(task any, input any, output any) error {
	name := getFunctionName(task)

	step := &allure.Step{
		Name:  fmt.Sprintf("Task(%s)", name),
		Start: workflow.Now(t.ctx).UnixMilli(),
	}

	if t.step != nil {
		t.step.Steps = append(t.step.Steps, step)
	}

	if param, ok := newParameter("input", input); ok {
		step.Parameters = append(step.Parameters, param)
	}

	opts := DefaultActivityOptions

	if t.options != nil {
		opts = *t.options
	}

	// set Summary with test name
	opts.Summary = t.name

	ctx := workflow.WithActivityOptions(t.ctx, opts)

	future := workflow.ExecuteActivity(ctx, task, input)

	var err error

	defer func() {
		step.Stop = workflow.Now(t.ctx).UnixMilli()
		step.Status = allure.Passed

		if err != nil {
			step.Status = allure.Failed
			step.StatusDetails = allure.StatusDetail{
				Message: "task failed",
				Trace:   err.Error(),
			}
		}

		if param, ok := newParameter("output", output); ok {
			step.Parameters = append(step.Parameters, param)
		}
	}()

	err = workflow.Await(t.ctx, future.IsReady)
	if err != nil {
		return errors.Join(ErrTaskExecute, err)
	}

	err = future.Get(t.ctx, output)
	if err != nil {
		return errors.Join(ErrTaskResult, err)
	}

	return nil
}
