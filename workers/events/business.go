package events

import (
	"context"

	"go.temporal.io/sdk/activity"
)

// PublishBusiness emits a business event from within an activity. A nil
// Publisher is tolerated so activities stay runnable without NATS.
func PublishBusiness(ctx context.Context, pub Publisher, pattern, typ string, data any) {
	if pub == nil {
		return
	}
	info := activity.GetInfo(ctx)
	env := NewEnvelope(pattern, info.WorkflowExecution.ID, info.WorkflowExecution.RunID, typ, data)
	if err := pub.Publish(ctx, pattern, env); err != nil {
		activity.GetLogger(ctx).Warn("publish business event failed", "type", typ, "error", err)
	}
}

// PublishBusinessAs emits a business event using an explicit workflowID / runID
// on the envelope. Use this from a child-workflow activity so the event lands
// on the parent's NATS subject instead of the child's.
func PublishBusinessAs(ctx context.Context, pub Publisher, pattern, workflowID, runID, typ string, data any) {
	if pub == nil {
		return
	}
	env := NewEnvelope(pattern, workflowID, runID, typ, data)
	if err := pub.Publish(ctx, pattern, env); err != nil {
		activity.GetLogger(ctx).Warn("publish business event failed", "type", typ, "error", err)
	}
}
