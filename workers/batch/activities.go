package batch

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Activities groups the batch pattern activities. Fields can be used for
// dependency injection (HTTP clients, DB handles, event publisher, etc.).
type Activities struct {
	Publisher events.Publisher
	// FailureRate is the probability (0..1) that a stage fails on its first
	// attempt so Temporal must retry it. Zero disables injection (default for
	// tests). Applied per stage: with 4 stages a per-image failure probability
	// of ~18.5% maps to FailureRate ≈ 0.05.
	FailureRate float64
}

// ResizeImage is the first stage — simulated image resize.
func (a *Activities) ResizeImage(ctx context.Context, in StageInput) error {
	return a.runStage(ctx, in, false)
}

// CreateThumbnail is the second stage — simulated thumbnail generation.
func (a *Activities) CreateThumbnail(ctx context.Context, in StageInput) error {
	return a.runStage(ctx, in, false)
}

// UploadToCDN is the third stage — simulated CDN upload.
func (a *Activities) UploadToCDN(ctx context.Context, in StageInput) error {
	return a.runStage(ctx, in, false)
}

// WriteMetadata is the final stage — simulated metadata write. As the closing
// stage it also publishes the batch.item.completed event so the UI sees one
// completion per image, not per stage.
func (a *Activities) WriteMetadata(ctx context.Context, in StageInput) error {
	return a.runStage(ctx, in, true)
}

// runStage is the shared body for every pipeline stage. It publishes the
// started event, sleeps, optionally injects a first-attempt failure, and on
// the final stage (emitCompleted=true) publishes batch.item.completed. All
// business events route to the parent workflow's NATS subject via
// PublishBusinessAs so the UI sees one per-batch stream.
func (a *Activities) runStage(ctx context.Context, in StageInput, emitCompleted bool) error {
	info := activity.GetInfo(ctx)
	attempt := int(info.Attempt)
	runID := info.WorkflowExecution.RunID
	activity.GetLogger(ctx).Info("Processing stage",
		"batch", in.BatchID, "index", in.Index, "service", in.Service, "attempt", attempt)

	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.RootWorkflowID, runID, TypeItemStarted, map[string]any{
		"index":   in.Index,
		"service": in.Service,
		"attempt": attempt,
	})

	time.Sleep(400 * time.Millisecond)

	// Failure injection: only on the first attempt, so a bounded retry policy
	// always reaches a successful second attempt.
	if attempt == 1 && in.FailureRate > 0 && rand.Float64() < in.FailureRate {
		events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.RootWorkflowID, runID, TypeItemAttemptFailed, map[string]any{
			"index":   in.Index,
			"service": in.Service,
			"attempt": attempt,
			"error":   fmt.Sprintf("%s service timeout", in.Service),
		})
		return fmt.Errorf("%s service timeout", in.Service)
	}

	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.RootWorkflowID, runID, TypeItemStageCompleted, map[string]any{
		"index":   in.Index,
		"service": in.Service,
		"attempt": attempt,
	})

	if emitCompleted {
		events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.RootWorkflowID, runID, TypeItemCompleted, map[string]any{
			"index": in.Index,
		})
	}
	return nil
}

// ReportBatchSummary writes a final per-batch summary event. It is the single
// closing step of the workflow, emitted regardless of individual item outcomes.
// It runs from the parent workflow context so PublishBusiness naturally routes
// to the parent's subject.
func (a *Activities) ReportBatchSummary(ctx context.Context, result BatchResult) error {
	activity.GetLogger(ctx).Info("Reporting batch summary",
		"batch", result.BatchID, "total", result.Total,
		"processed", result.Processed, "failed", result.Failed)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeSummaryReported, map[string]any{
		"batchId":   result.BatchID,
		"total":     result.Total,
		"processed": result.Processed,
		"failed":    result.Failed,
	})
	return nil
}
