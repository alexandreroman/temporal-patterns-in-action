// Package batch implements the long-running batch pattern: dispatch N child
// workflows in one shot, then report a summary. Each child workflow fans out
// to the 4 pipeline stages. The effective sliding window is enforced by the
// worker's MaxConcurrentActivityExecutionSize rather than an in-workflow
// semaphore. Retries are bounded (MaximumAttempts=3) so a transient stage
// timeout is retried to success.
package batch

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Service names are one per stage: each child workflow walks the full
// pipeline (resize, thumbnail, cdn, metadata) in order, so the Service on
// every StageInput matches the stage doing the work.
const (
	serviceResize    = "resize"
	serviceThumbnail = "thumbnail"
	serviceCDN       = "cdn"
	serviceMetadata  = "metadata"
)

// stageActivityOptions is the retry/timeout policy shared by the 4 pipeline
// stages inside ProcessImageWorkflow.
func stageActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    3,
		},
	}
}

// BatchProcessingWorkflow dispatches Total child workflows in one shot, then
// reports a summary. The effective sliding window is enforced by the worker's
// MaxConcurrentActivityExecutionSize rather than in-workflow logic. Individual
// item failures are counted and reported — they never fail the workflow itself.
func BatchProcessingWorkflow(ctx workflow.Context, input BatchInput) (BatchResult, error) {
	logger := workflow.GetLogger(ctx)
	rootID := workflow.GetInfo(ctx).WorkflowExecution.ID

	// Activity options for the closing summary. Child workflows set their own
	// options on the stage activities.
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	result := BatchResult{BatchID: input.BatchID, Total: input.Total}
	progress := Progress{Total: input.Total}

	if err := workflow.SetQueryHandler(ctx, "getProgress", func() (Progress, error) {
		return progress, nil
	}); err != nil {
		return result, err
	}

	futures := make([]workflow.Future, 0, input.Total)

	// Dispatch loop — start every child immediately. The worker's
	// MaxConcurrentActivityExecutionSize caps how many stage activities run in
	// parallel, which throttles the effective sliding window.
	for i := 0; i < input.Total; i++ {
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s-item-%03d", rootID, i),
		})
		in := ImageInput{
			BatchID:        input.BatchID,
			RootWorkflowID: rootID,
			Index:          i,
			FailureRate:    input.FailureRate,
		}
		progress.InFlight++
		future := workflow.ExecuteChildWorkflow(childCtx, ProcessImageWorkflow, in)
		futures = append(futures, future)
	}

	// Drain: wait for every child and update counters. Counters live on the
	// workflow goroutine so the query handler sees a consistent snapshot.
	for _, f := range futures {
		if err := f.Get(ctx, nil); err != nil {
			progress.Failed++
			result.Failed++
			logger.Warn("item failed after retries", "error", err)
		} else {
			progress.Processed++
			result.Processed++
		}
		progress.InFlight--
	}

	var a *Activities
	if err := workflow.ExecuteActivity(ctx, a.ReportBatchSummary, result).Get(ctx, nil); err != nil {
		return result, err
	}
	return result, nil
}

// ProcessImageWorkflow runs the 4 pipeline stages sequentially for a single
// image. Returning an error from any stage stops the pipeline and surfaces the
// failure to the parent, which counts it as a failed item without aborting the
// batch.
func ProcessImageWorkflow(ctx workflow.Context, in ImageInput) error {
	ctx = workflow.WithActivityOptions(ctx, stageActivityOptions())

	// Each stage carries its own service name so UI labels and failure
	// messages line up with the stage doing the work.
	var a *Activities
	stages := []struct {
		activity any
		service  string
	}{
		{a.ResizeImage, serviceResize},
		{a.CreateThumbnail, serviceThumbnail},
		{a.UploadToCDN, serviceCDN},
		{a.WriteMetadata, serviceMetadata},
	}
	for _, s := range stages {
		stage := StageInput{
			BatchID:        in.BatchID,
			RootWorkflowID: in.RootWorkflowID,
			Index:          in.Index,
			Service:        s.service,
			FailureRate:    in.FailureRate,
		}
		if err := workflow.ExecuteActivity(ctx, s.activity, stage).Get(ctx, nil); err != nil {
			return err
		}
	}
	return nil
}
