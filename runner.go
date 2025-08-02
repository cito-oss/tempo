package tempo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/ozontech/allure-go/pkg/allure"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

const reportStepField = "allure_step"
const reportReplaceFlag = "_report_child_workflow_"

var reportReplaceID = func(workflowID string, runID string) string {
	return fmt.Sprintf("%s||%s", workflowID, runID)
}

type Option func(*runConfig)

type runConfig struct {
	reporting     bool
	reportHandler func(report *allure.Result)
}

func WithReporting(enabled bool) Option {
	return func(cfg *runConfig) {
		cfg.reporting = enabled
	}
}

func WithReportHandler(fn func(report *allure.Result)) Option {
	return func(cfg *runConfig) {
		cfg.reportHandler = fn
	}
}

func Run(cli client.Client, queue string, id string, plan Plan, output any, opts ...Option) error {
	cfg := &runConfig{}

	for _, opt := range opts {
		opt(cfg)
	}

	// TODO expose this to outside control
	workflowOpts := client.StartWorkflowOptions{
		ID:                       id,
		TaskQueue:                queue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}

	var run client.WorkflowRun
	var workflowerr error

	if cfg.reporting {
		result := allure.NewResult(plan.name, id)

		if param, ok := newParameter("input", plan.input); ok {
			result.Parameters = append(result.Parameters, param)
		}

		result.Start = allure.GetNow()

		defer func() {
			result.Stop = allure.GetNow()

			result.Status = allure.Passed // assume tests passed...

			if workflowerr != nil {
				result.Status = allure.Failed // ... but set to failed if the workflow returned error
			}

			if param, ok := newParameter("output", output); ok {
				result.Parameters = append(result.Parameters, param)
			}

			log.Println("saving report...")

			desc, err := cli.DescribeWorkflowExecution(context.Background(), run.GetID(), run.GetRunID())
			if err != nil {
				log.Printf("report: fail to describe parent workflow: %s", err)
			}

			if field, ok := desc.GetWorkflowExecutionInfo().GetMemo().GetFields()[reportStepField]; ok {
				dc := converter.GetDefaultDataConverter()

				step := &allure.Step{}

				err = dc.FromPayload(field, step)
				if err != nil {
					log.Printf("report: fail to decode parent memo field: %s", err)
				} else {
					result.Steps = append(result.Steps, step)
				}
			}

			req := &workflowservice.ListWorkflowExecutionsRequest{
				Query: fmt.Sprintf(`ParentWorkflowId='%s'`, run.GetID()),
			}

			resp, err := cli.ListWorkflow(context.Background(), req)
			if err != nil {
				log.Printf("report: fail to list child workflows: %s", err)
			}

			// get all child workflows (if any) and parse their `reportStepField`
			// into the map below
			childs := map[string]*allure.Step{}
			for _, exec := range resp.GetExecutions() {
				if field, ok := exec.GetMemo().GetFields()[reportStepField]; ok {
					dc := converter.GetDefaultDataConverter()

					step := &allure.Step{}

					err := dc.FromPayload(field, step)
					if err != nil {
						log.Printf("report: fail to decode child memo field: %s", err)
						continue
					}

					id := reportReplaceID(exec.GetExecution().GetWorkflowId(), exec.GetExecution().GetRunId())
					childs[id] = step
				}
			}

			mergeSteps(childs, result.Steps)
			decodeParameters(result.Parameters, result.Steps)

			if cfg.reportHandler == nil {
				err = result.Print()
				if err != nil {
					log.Printf("report: failed to save: %s", err)
				}
			} else {
				cfg.reportHandler(result)
			}
		}()
	}

	run, workflowerr = cli.ExecuteWorkflow(context.Background(), workflowOpts, plan.name, plan.input)
	if workflowerr != nil {
		return errors.Join(ErrWorkflowExecute, workflowerr)
	}

	workflowerr = run.Get(context.Background(), output)
	if workflowerr != nil {
		return errors.Join(ErrWorkflowResult, workflowerr)
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
	client    client.Client
	limit     chan struct{}
	queue     string
	plans     []Plan
	reporting bool
	handler   func(result *allure.Result)
}

func (r *Runner) SetReporting(enabled bool) {
	r.reporting = enabled
}

func (r *Runner) SetResultHandler(fn func(result *allure.Result)) {
	r.handler = fn
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
	var opts []Option

	if r.reporting {
		opts = append(opts, WithReporting(true))
	}

	if r.handler != nil {
		opts = append(opts, WithReportHandler(r.handler))
	}

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

			err := Run(r.client, r.queue, id, fn, nil, opts...)
			if err != nil {
				errs <- errors.Join(ErrPlanRun, err)
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

// mergeSteps recursively walks through steps to merge all child workflow steps into main result.
func mergeSteps(childs map[string]*allure.Step, steps []*allure.Step) {
	for _, step := range steps {
		for _, attachment := range step.Attachments {
			if attachment.Name == reportReplaceFlag {
				id := attachment.Source

				done, ok := childs[id]
				if ok {
					step.Steps = append(step.Steps, done)
				} else {
					log.Printf("report: missing child workflow report: %s", id)
				}
			}
		}

		step.Attachments = nil

		if len(step.Steps) > 0 {
			mergeSteps(childs, step.Steps)
		}
	}
}

func decodeParameters(params []*allure.Parameter, steps []*allure.Step) {
	for _, param := range params {
		if str, ok := param.Value.(string); ok {
			var data any

			err := json.Unmarshal([]byte(str), &data)
			if err != nil {
				continue
			}

			param.Value = data
		}
	}

	for _, step := range steps {
		if len(step.Parameters) > 0 || len(step.Steps) > 0 {
			decodeParameters(step.Parameters, step.Steps)
		}
	}
}
