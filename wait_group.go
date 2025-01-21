package tempo

import "go.temporal.io/sdk/workflow"

type WaitGroup struct {
	ctx workflow.Context
	workflow.WaitGroup
}

func (wg *WaitGroup) Wait() {
	wg.WaitGroup.Wait(wg.ctx)
}
