package batch

import (
	"context"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "batch"

// Batch-specific business event types. Prefixed with the pattern name to
// guarantee no collision with types emitted by other patterns.
const (
	TypeItemStarted       = "batch.item.started"
	TypeItemCompleted     = "batch.item.completed"
	TypeItemAttemptFailed = "batch.item.attempt_failed"
	TypeSummaryReported   = "batch.summary.reported"
)

// publishBusiness emits a business event from within an activity. A nil
// Publisher is tolerated so the activities stay runnable without NATS.
func (a *Activities) publishBusiness(ctx context.Context, typ string, data any) {
	if a.Publisher == nil {
		return
	}
	info := activity.GetInfo(ctx)
	env := events.NewEnvelope(Pattern,
		info.WorkflowExecution.ID, info.WorkflowExecution.RunID, typ, data)
	if err := a.Publisher.Publish(ctx, Pattern, env); err != nil {
		activity.GetLogger(ctx).Warn("publish business event failed",
			"type", typ, "error", err)
	}
}
