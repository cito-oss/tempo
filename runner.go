package tempo

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

func Run(cli client.Client, queue string, id string, plan Plan, output any) error {
	// TODO expose this to outside control
	opts := client.StartWorkflowOptions{
		ID:                       id,
		TaskQueue:                queue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}

	run, err := cli.ExecuteWorkflow(context.Background(), opts, plan.name, plan.input)
	if err != nil {
		return errors.Join(ErrWorkflowExecute, err)
	}

	err = run.Get(context.Background(), output)
	if err != nil {
		return errors.Join(ErrWorkflow, err)
	}

	return nil
}

func NewRunner(client client.Client, queue string, plans ...Plan) *Runner {
	return &Runner{
		client: client,
		queue:  queue,
		plans:  plans,
	}
}

type Runner struct {
	client client.Client
	limit  chan struct{}
	queue  string
	plans  []Plan
}

func (r *Runner) SetLimit(limit int) {
	if limit < 0 {
		r.limit = nil
		return
	}
	if len(r.limit) != 0 {
		panic(ErrSetLimitWithGoroutines)
	}
	r.limit = make(chan struct{}, limit)
}

func (r *Runner) Run(prefix string) error {
	var total = len(r.plans)

	errs := make(chan error, total)

	var wg sync.WaitGroup
	wg.Add(total)

	for _, plan := range r.plans {
		go func(fn Plan) {
			if r.limit != nil {
				r.limit <- struct{}{}
			}

			defer func() {
				if r.limit != nil {
					<-r.limit
				}
				wg.Done()
			}()

			id := fmt.Sprintf("%s@%s", prefix, fn.name)

			err := Run(r.client, r.queue, id, fn, nil)
			if err != nil {
				errs <- errors.Join(ErrTest, err)
			}
		}(plan)
	}

	wg.Wait()
	close(errs)

	if len(errs) == 0 {
		return nil
	}

	var err error

	for e := range errs {
		err = errors.Join(err, e)
	}

	return err
}
