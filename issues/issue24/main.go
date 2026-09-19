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

// PassingTest passes. It only has to be inside a t.Run, parked in a t.Task, when
// the eviction lands.
func PassingTest(t *tempo.T) {
	t.SetActivityOptions(workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})

	t.Run("subtest parked in a task", func(t *tempo.T) {
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
