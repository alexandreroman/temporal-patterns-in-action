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
	// FailureRate is the probability (0..1) that an item fails on its first
	// attempt so Temporal must retry it. Zero disables injection (default for
	// tests).
	FailureRate float64
}

// ProcessImage is the per-item activity. It sleeps to simulate work and
// optionally injects a first-attempt retryable failure driven by FailureRate.
func (a *Activities) ProcessImage(ctx context.Context, item ImageItem) error {
	attempt := int(activity.GetInfo(ctx).Attempt)
	activity.GetLogger(ctx).Info("Processing image",
		"batch", item.BatchID, "index", item.Index, "service", item.Service, "attempt", attempt)

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeItemStarted, map[string]any{
		"index":   item.Index,
		"service": item.Service,
		"attempt": attempt,
	})

	time.Sleep(1500 * time.Millisecond)

	// Failure injection: only on the first attempt, so a bounded retry policy
	// always reaches a successful second attempt.
	if attempt == 1 && a.FailureRate > 0 && rand.Float64() < a.FailureRate {
		events.PublishBusiness(ctx, a.Publisher, Pattern, TypeItemAttemptFailed, map[string]any{
			"index":   item.Index,
			"service": item.Service,
			"attempt": attempt,
			"error":   fmt.Sprintf("%s service timeout", item.Service),
		})
		return fmt.Errorf("%s service timeout", item.Service)
	}

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeItemCompleted, map[string]any{
		"index":   item.Index,
		"service": item.Service,
	})
	return nil
}

// ReportBatchSummary writes a final per-batch summary event. It is the single
// closing step of the workflow, emitted regardless of individual item outcomes.
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
