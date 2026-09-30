// Package batch implements the long-running batch pattern: process N images
// through child workflows while a sliding window in the parent keeps at most
// windowSize children in flight, then report a summary. Each child runs the 4
// pipeline stages sequentially. Retries are bounded (MaximumAttempts=3) so a
// transient stage timeout is retried to success.
package batch

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// windowSize is the maximum number of child workflows in flight at once. It
// matches PARALLELISM in frontend/app/pages/patterns/batch.vue.
const windowSize = 4

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

// BatchProcessingWorkflow processes Total images with a sliding window: it
// starts a new child workflow only when one of the windowSize in-flight
// children finishes, then drains the window and reports a summary. Individual
// item failures are counted and reported — they never fail the workflow itself.
func BatchProcessingWorkflow(ctx workflow.Context, input BatchInput) (BatchResult, error) {
	logger := workflow.GetLogger(ctx)
	info := workflow.GetInfo(ctx)
	rootID := info.WorkflowExecution.ID
	rootRunID := info.WorkflowExecution.RunID

	// Activity options for the closing summary. Child workflows set their own
	// options on the stage activities.
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	result := BatchResult{BatchID: input.BatchID, Total: input.Total}

	// Every child future is added to the selector; each Select call runs the
	// callback of exactly one finished child.
	selector := workflow.NewSelector(ctx)
	inFlight := 0

	// Dispatch loop — the sliding window. Once windowSize children are in
	// flight, wait for one to finish before starting the next.
	for i := range input.Total {
		if inFlight == windowSize {
			selector.Select(ctx) // wait for a child to finish, freeing a slot
			inFlight--
		}

		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s-item-%03d", rootID, i),
		})
		in := ImageInput{
			BatchID:        input.BatchID,
			RootWorkflowID: rootID,
			RootRunID:      rootRunID,
			Index:          i,
			FailureRate:    input.FailureRate,
		}
		future := workflow.ExecuteChildWorkflow(childCtx, ProcessImageWorkflow, in)
		selector.AddFuture(future, func(f workflow.Future) {
			if err := f.Get(ctx, nil); err != nil {
				result.Failed++
				logger.Warn("item failed after retries", "error", err)
			} else {
				result.Processed++
			}
		})
		inFlight++
	}

	// Drain the last children still in the window.
	for ; inFlight > 0; inFlight-- {
		selector.Select(ctx)
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
			RootRunID:      in.RootRunID,
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
