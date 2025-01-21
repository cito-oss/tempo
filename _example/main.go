package main

import (
	"context"
	"time"

	"github.com/cito-oss/tempo"
	"github.com/cito-oss/tempo/_example/subtests"
	"github.com/cito-oss/tempo/_example/tasks"
	"github.com/cito-oss/tempo/_example/tests"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"golang.org/x/sync/errgroup"
)

func main() {
	Example1()
	Example2()
}

func Example1() {
	cli, err := client.Dial(client.Options{})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	queue := "default"

	// WORKER
	myworker := worker.New(cli, queue, worker.Options{})

	tempo.Worker(myworker, tempo.Registry{
		Tests: []tempo.Test{
			tempo.NewTest(tests.SayHelloInParallel),
			tempo.NewTest(tests.SayHelloToEveryone),
			tempo.NewTest(tests.SayHelloWithSubtest),
			tempo.NewTest(tests.SayHelloWorld),
			tempo.NewTest(tests.UnknownPerson),
			tempo.NewTestWithInput(tests.SayHello),
			tempo.NewTestWithOutput(subtests.QuoteOfTheDay),
			tempo.NewTestWithInputAndOutput(subtests.SayHelloToPerson),
		},
		Tasks: []tempo.Task{
			tasks.Greeter,
			tasks.PersonAge,
			tasks.RandomQuote,
		},
	})

	err = myworker.Start()
	if err != nil {
		panic(err)
	}
	defer myworker.Stop()
	// /WORKER

	// RUNNER
	myrunner := tempo.NewRunner(cli, queue,
		tempo.NewPlan(tests.SayHello, "John Doe"),
		tempo.NewPlan(tests.SayHelloInParallel, nil),
		tempo.NewPlan(tests.SayHelloToEveryone, nil),
		tempo.NewPlan(tests.SayHelloWithSubtest, nil),
		tempo.NewPlan(tests.SayHelloWorld, nil),
		tempo.NewPlan(tests.UnknownPerson, nil),
	)

	err = myrunner.Run("v1.0.0-" + time.Now().Format("20060102T150405"))
	if err != nil {
		panic(err)
	}
	// /RUNNER

	// alternatively, if you don't want to use the Runner, you can call directly
	// the tempo.Run, like in the example below, or even make your own implementation (see Example2 below).
	//
	// plan := tempo.NewPlan(tests.SayHello, "John Doe")
	// err := tempo.Run(cli, queue, "SayHelloToJohnDoe", plan, nil)
}

func Example2() {
	cli, err := client.Dial(client.Options{})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	queue := "default"

	// WORKER
	list := []tempo.Test{
		tempo.NewTest(tests.SayHelloInParallel),
		tempo.NewTest(tests.SayHelloToEveryone),
		tempo.NewTest(tests.SayHelloWithSubtest),
		tempo.NewTest(tests.SayHelloWorld),
		tempo.NewTest(tests.UnknownPerson),
		tempo.NewTestWithInput(tests.SayHello),
		tempo.NewTestWithOutput(subtests.QuoteOfTheDay),
		tempo.NewTestWithInputAndOutput(subtests.SayHelloToPerson),
	}

	myworker := worker.New(cli, queue, worker.Options{})

	for _, item := range list {
		// register all tests
		myworker.RegisterWorkflowWithOptions(
			item.Function(),
			workflow.RegisterOptions{Name: item.Name()},
		)
	}

	myworker.RegisterActivity(tasks.Greeter)
	myworker.RegisterActivity(tasks.PersonAge)
	myworker.RegisterActivity(tasks.RandomQuote)

	err = myworker.Start()
	if err != nil {
		panic(err)
	}

	defer myworker.Stop()
	// /WORKER

	// RUNNER
	var g errgroup.Group

	plans := []tempo.Plan{
		tempo.NewPlan(tests.SayHello, "John Doe"),
		tempo.NewPlan(tests.SayHelloInParallel, nil),
		tempo.NewPlan(tests.SayHelloToEveryone, nil),
		tempo.NewPlan(tests.SayHelloWithSubtest, nil),
		tempo.NewPlan(tests.SayHelloWorld, nil),
		tempo.NewPlan(tests.UnknownPerson, nil),
	}

	for _, plan := range plans {
		g.Go(func() error {
			run, err := cli.ExecuteWorkflow(
				context.Background(),
				client.StartWorkflowOptions{
					ID:        plan.Name() + "@" + time.Now().Format("20060102T150405"),
					TaskQueue: queue,
				},
				plan.Name(),
				plan.Input(),
			)
			if err != nil {
				return err
			}

			err = run.Get(context.Background(), nil)
			if err != nil {
				return err
			}

			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		panic(err)
	}
	// /RUNNER
}
