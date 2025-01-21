package tempo

import (
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type Registry struct {
	Tests []Test
	Tasks []Task
}

func Worker(w worker.Worker, reg Registry) {
	for _, item := range reg.Tests {
		// TODO expose this to outside control
		opts := workflow.RegisterOptions{
			Name: item.Name(),
		}

		w.RegisterWorkflowWithOptions(item.Function(), opts)
	}

	for _, item := range reg.Tasks {
		w.RegisterActivity(item)
	}
}
