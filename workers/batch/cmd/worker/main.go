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
		w.RegisterWorkflow(batch.ProcessImageWorkflow)

		a := &batch.Activities{Publisher: pub}
		w.RegisterActivityWithOptions(a.ResizeImage, activity.RegisterOptions{Name: "resize-image"})
		w.RegisterActivityWithOptions(a.CreateThumbnail, activity.RegisterOptions{Name: "create-thumbnail"})
		w.RegisterActivityWithOptions(a.UploadToCDN, activity.RegisterOptions{Name: "upload-cdn"})
		w.RegisterActivityWithOptions(a.WriteMetadata, activity.RegisterOptions{Name: "write-metadata"})
		w.RegisterActivityWithOptions(a.ReportBatchSummary, activity.RegisterOptions{Name: "report-batch-summary"})
	}, worker.Options{}) // no activity cap: the workflow's sliding window bounds the load
}
