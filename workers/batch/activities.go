package batch

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

const itemHalfSleep = 700 * time.Millisecond

// Activities groups the batch pattern activities. Fields can be used for
// dependency injection (HTTP clients, DB handles, event publisher, etc.).
type Activities struct {
	Publisher events.Publisher
	// FailureRate is the probability (0..1) that an item fails on its first
	// attempt so Temporal must retry it. Zero disables injection (default for
	// tests).
	FailureRate float64
}

// ProcessImage is the long-running per-item activity. It sleeps in two halves
// with a heartbeat in between so the demo shows real heartbeat traffic, and
// optionally injects a first-attempt retryable failure driven by FailureRate.
func (a *Activities) ProcessImage(ctx context.Context, item ImageItem) error {
	attempt := int(activity.GetInfo(ctx).Attempt)
	activity.GetLogger(ctx).Info("Processing image",
		"batch", item.BatchID, "index", item.Index, "service", item.Service, "attempt", attempt)

	a.publishBusiness(ctx, TypeItemStarted, map[string]any{
		"index":   item.Index,
		"service": item.Service,
		"attempt": attempt,
	})

	start := time.Now()
	time.Sleep(itemHalfSleep)
	activity.RecordHeartbeat(ctx, item.Index)
	time.Sleep(itemHalfSleep)

	// Failure injection: only on the first attempt, so a bounded retry policy
	// always reaches a successful second attempt.
	if attempt == 1 && a.FailureRate > 0 && rand.Float64() < a.FailureRate {
		a.publishBusiness(ctx, TypeItemAttemptFailed, map[string]any{
			"index":   item.Index,
			"service": item.Service,
			"attempt": attempt,
			"error":   fmt.Sprintf("%s service timeout", item.Service),
		})
		return fmt.Errorf("%s service timeout", item.Service)
	}

	a.publishBusiness(ctx, TypeItemCompleted, map[string]any{
		"index":      item.Index,
		"service":    item.Service,
		"durationMs": time.Since(start).Milliseconds(),
	})
	return nil
}

// ReportBatchSummary writes a final per-batch summary event. It is the single
// closing step of the workflow, emitted regardless of individual item outcomes.
func (a *Activities) ReportBatchSummary(ctx context.Context, result BatchResult) error {
	activity.GetLogger(ctx).Info("Reporting batch summary",
		"batch", result.BatchID, "total", result.Total,
		"processed", result.Processed, "failed", result.Failed)
	a.publishBusiness(ctx, TypeSummaryReported, map[string]any{
		"batchId":   result.BatchID,
		"total":     result.Total,
		"processed": result.Processed,
		"failed":    result.Failed,
	})
	return nil
}
