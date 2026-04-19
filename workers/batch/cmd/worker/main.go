// Package main runs the long-running batch pattern worker.
package main

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/batch"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func main() {
	events.RunWorker(batch.Pattern, batch.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(batch.BatchProcessingWorkflow)

		// Register each batch activity under its canonical kebab-case name so the
		// NATS event interceptor emits progress.step.* events matching the step IDs
		// used by the workflow and the frontend pipeline.
		a := &batch.Activities{Publisher: pub, FailureRate: 0.12}
		w.RegisterActivityWithOptions(a.ProcessImage, activity.RegisterOptions{Name: "process-image"})
		w.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})
	})
}
