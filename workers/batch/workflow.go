// Package batch implements the long-running batch pattern: dispatch N activities
// with a sliding window of at most `Parallelism` in flight at any time, then
// report a summary. Retries are bounded (MaximumAttempts=3) so a transient
// service timeout is retried to success.
package batch

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// services is the round-robin pool of downstream services each item is routed
// to. The index of the item picks the service via `services[i % len(services)]`.
var services = []string{"resize", "thumbnail", "cdn", "metadata"}

// BatchProcessingWorkflow dispatches Total items with at most Parallelism in
// flight at once, then reports a summary. Individual item failures are counted
// and reported — they never fail the workflow itself.
func BatchProcessingWorkflow(ctx workflow.Context, input BatchInput) (BatchResult, error) {
	logger := workflow.GetLogger(ctx)

	parallelism := input.Parallelism
	if parallelism < 1 {
		parallelism = 1
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		HeartbeatTimeout:    5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := BatchResult{BatchID: input.BatchID, Total: input.Total}
	progress := Progress{Total: input.Total}

	if err := workflow.SetQueryHandler(ctx, "getProgress", func() (Progress, error) {
		return progress, nil
	}); err != nil {
		return result, err
	}

	var a *Activities
	sem := workflow.NewSemaphore(ctx, int64(parallelism))
	futures := make([]workflow.Future, 0, input.Total)

	// Dispatch loop — acquire a slot before starting each activity so at most
	// `parallelism` are in flight. A workflow.Go goroutine releases the slot
	// as soon as the activity future resolves.
	for i := 0; i < input.Total; i++ {
		if err := sem.Acquire(ctx, 1); err != nil {
			return result, err
		}
		item := ImageItem{
			BatchID: input.BatchID,
			Index:   i,
			Service: services[i%len(services)],
		}
		progress.InFlight++
		future := workflow.ExecuteActivity(ctx, a.ProcessImage, item)
		futures = append(futures, future)

		workflow.Go(ctx, func(gctx workflow.Context) {
			defer sem.Release(1)
			_ = future.Get(gctx, nil)
		})
	}

	// Drain: wait for every activity and update counters. Counters live on the
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

	if err := workflow.ExecuteActivity(ctx, a.ReportBatchSummary, result).Get(ctx, nil); err != nil {
		return result, err
	}
	return result, nil
}
