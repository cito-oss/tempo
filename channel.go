package tempo

import "go.temporal.io/sdk/workflow"

// Channel is a wrapper for Temporal Channel
type Channel struct {
	ctx   workflow.Context
	chann workflow.Channel
}

func (c *Channel) Send(v any) {
	c.chann.Send(c.ctx, v)
}

func (c *Channel) Receive(valuePtr any) (more bool) {
	return c.chann.Receive(c.ctx, valuePtr)
}

func (c *Channel) Close() {
	c.chann.Close()
}
