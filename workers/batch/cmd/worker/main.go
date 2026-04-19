// Package main runs the long-running batch pattern worker.
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/batch"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func main() {
	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		address = "localhost:7233"
	}

	natsURL, ok := os.LookupEnv("NATS_URL")
	if !ok {
		natsURL = "nats://localhost:4222"
	}

	publisher, err := events.NewPublisher(natsURL)
	if err != nil {
		log.Printf("nats unavailable at %s (%v) — running without event publishing", natsURL, err)
		publisher = events.NopPublisher{}
	}
	defer publisher.Close()

	c, err := client.Dial(client.Options{HostPort: address})
	if err != nil {
		log.Fatalf("unable to create temporal client: %v", err)
	}
	defer c.Close()

	w := worker.New(c, batch.TaskQueue, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{
			events.NewInterceptor(publisher, batch.Pattern),
		},
	})
	w.RegisterWorkflow(batch.BatchProcessingWorkflow)

	// Register each batch activity under its canonical kebab-case name so the
	// NATS event interceptor emits progress.step.* events matching the step IDs
	// used by the workflow and the frontend pipeline.
	a := &batch.Activities{Publisher: publisher, FailureRate: 0.12}
	w.RegisterActivityWithOptions(a.ProcessImage, activity.RegisterOptions{Name: "process-image"})
	w.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})

	log.Printf("batch worker connected to %s — listening on task queue: %s", address, batch.TaskQueue)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}
