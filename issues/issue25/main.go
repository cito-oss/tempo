// Command bug reproduces the panic that survives the #24 fix.
//
// Run it with a Temporal server on localhost:7233, against this checkout:
//
//	temporal server start-dev
//	go run main.go
//
// It crashes:
//
//	panic: getState: illegal access from outside of workflow context
//	  go.temporal.io/sdk/internal.NewWaitGroup   workflow.go:860
//	  github.com/cito-oss/tempo.(*T).child       t.go:52
//	  github.com/cito-oss/tempo.(*T).cleanup     t.go:153
//	  github.com/cito-oss/tempo.(*T).Run         t.go:189
//
// Same cause as #24 — the SDK evicts a workflow execution from the sticky cache,
// dispatcher.Close() sends runtime.Goexit() into the parked coroutine, and tempo's
// deferred teardown then touches a workflow context whose dispatcher has stopped
// executing. #24 guarded one such call, the wg.Wait inside stop(). This is a
// different one, reached a frame earlier:
//
//	func (t *T) cleanup() {
//		if len(t.cleanups) == 0 {
//			return                                  // why #24 landed in stop()
//		}
//		child := t.child(...)                       // -> workflow.NewWaitGroup -> panic
//	}
//
// So the trigger is simply a subtest that registers a cleanup. The cleanup body
// never runs and does not need to do anything — registering it is enough, because
// T.child builds a workflow.WaitGroup unconditionally.
//
// As in #24 the panic is on a worker goroutine, so it takes the whole process down
// and every other test in flight with it, and the test being run here passes.
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cito-oss/tempo"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	endpoint  = "localhost:7233"
	namespace = "default"

	// More workflows than cache slots, so the worker has to evict one that is
	// parked inside SlowTask.
	workflows = 4
)

// SlowTask keeps the test parked long enough to be evicted while it waits.
func SlowTask(ctx context.Context) error {
	time.Sleep(2 * time.Second)

	return nil
}

// PassingTest passes. It only has to hold a cleanup and be parked in a t.Task when
// the eviction lands.
func PassingTest(t *tempo.T) {
	t.SetActivityOptions(workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})

	t.Run("subtest parked in a task", func(t *tempo.T) {
		// The whole trigger. An empty body is enough: cleanup() only has to get
		// past its len(t.cleanups) == 0 guard to build a child T.
		t.Cleanup(func() {})

		err := t.Task(SlowTask, nil, nil)
		require.NoError(t, err)
	})
}

func main() {
	// One slot, so every other workflow in flight has to be evicted to make room.
	// This only forces the eviction a loaded worker does on its own.
	worker.SetStickyWorkflowCacheSize(1)

	c, err := client.Dial(client.Options{HostPort: endpoint, Namespace: namespace})
	if err != nil {
		log.Fatalf("dial %s: %s", endpoint, err)
	}

	defer c.Close()

	queue := fmt.Sprintf("bug-%d", time.Now().UnixNano())

	w := worker.New(c, queue, worker.Options{})

	tempo.Worker(w, tempo.Registry{
		Tests: []tempo.Test{tempo.NewTest(PassingTest)},
		Tasks: []tempo.Task{SlowTask},
	})

	if err := w.Start(); err != nil {
		log.Fatalf("start worker: %s", err)
	}

	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Printf("running %d workflows on %s", workflows, queue)

	var wg sync.WaitGroup

	for i := range workflows {
		wg.Go(func() {
			run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
				ID:        fmt.Sprintf("%s-%d", queue, i),
				TaskQueue: queue,
			}, "PassingTest")
			if err != nil {
				log.Printf("workflow %d: start: %s", i, err)

				return
			}

			log.Printf("workflow %d: %v", i, run.Get(ctx, nil))
		})
	}

	wg.Wait()

	log.Print("worker survived: no panic, the bug is fixed")
}
