// Package main runs the long-running batch pattern worker.
package main

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/batch"
	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// maxConcurrentActivities replaces the previous in-workflow semaphore: the
// worker processes at most this many stage activities concurrently across all
// children, which throttles the effective sliding window of the batch.
const maxConcurrentActivities = 4

func main() {
	events.RunWorker(batch.Pattern, batch.TaskQueue, func(w worker.Worker, pub events.Publisher) {
		w.RegisterWorkflow(batch.BatchProcessingWorkflow)
		w.RegisterWorkflow(batch.ProcessImageWorkflow)

		// FailureRate is tuned so the per-image failure probability
		// (1-(1-r)^4) stays close to ~18.5%: 1-(1-0.05)^4.
		a := &batch.Activities{Publisher: pub, FailureRate: 0.05}
		w.RegisterActivityWithOptions(a.ResizeImage, activity.RegisterOptions{Name: "resize-image"})
		w.RegisterActivityWithOptions(a.CreateThumbnail, activity.RegisterOptions{Name: "create-thumbnail"})
		w.RegisterActivityWithOptions(a.UploadToCDN, activity.RegisterOptions{Name: "upload-cdn"})
		w.RegisterActivityWithOptions(a.WriteMetadata, activity.RegisterOptions{Name: "write-metadata"})
		w.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})
	}, worker.Options{MaxConcurrentActivityExecutionSize: maxConcurrentActivities})
}
