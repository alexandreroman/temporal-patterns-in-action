package events

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// NewInterceptor returns a WorkerInterceptor that publishes framework-level
// progression events for every workflow and activity hosted by the worker.
//
// Workflow-side events go through the PublishEvent local activity to preserve
// determinism; activity-side events publish directly to NATS.
func NewInterceptor(publisher Publisher, pattern string) interceptor.WorkerInterceptor {
	return &workerInterceptor{publisher: publisher, pattern: pattern}
}

type workerInterceptor struct {
	interceptor.WorkerInterceptorBase
	publisher Publisher
	pattern   string
}

func (w *workerInterceptor) InterceptActivity(
	_ context.Context, next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	i := &activityInbound{root: w}
	i.Next = next
	return i
}

func (w *workerInterceptor) InterceptWorkflow(
	_ workflow.Context, next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	i := &workflowInbound{root: w}
	i.Next = next
	return i
}

// workflowInbound publishes workflow.started / completed / failed events.
type workflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	root *workerInterceptor
}

func (w *workflowInbound) ExecuteWorkflow(
	ctx workflow.Context, in *interceptor.ExecuteWorkflowInput,
) (any, error) {
	info := workflow.GetInfo(ctx)
	start := workflow.Now(ctx)

	publishFromWorkflow(ctx, w.root.pattern, info,
		TypeWorkflowStarted, struct{}{})

	result, err := w.Next.ExecuteWorkflow(ctx, in)

	end := workflow.Now(ctx)
	durationMs := end.Sub(start).Milliseconds()
	if err != nil {
		publishFromWorkflow(ctx, w.root.pattern, info,
			TypeWorkflowFailed, map[string]any{"error": err.Error()})
	} else {
		publishFromWorkflow(ctx, w.root.pattern, info,
			TypeWorkflowCompleted, map[string]any{"durationMs": durationMs})
	}
	return result, err
}

// publishFromWorkflow fires a PublishEvent local activity and waits for its
// completion on a short timeout. Failures are logged but never surfaced —
// observability must never break the workflow.
func publishFromWorkflow(
	ctx workflow.Context, pattern string, info *workflow.Info, typ string, data any,
) {
	env := NewWorkflowEnvelope(
		pattern, info.WorkflowExecution.ID, info.WorkflowExecution.RunID,
		typ, workflow.Now(ctx), data,
	)
	lao := workflow.LocalActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	lctx := workflow.WithLocalActivityOptions(ctx, lao)
	if err := workflow.ExecuteLocalActivity(lctx, PublishEventActivityName, env).
		Get(lctx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("publish event failed",
			"type", typ, "error", err)
	}
}

// PublishFromWorkflow exposes the workflow-side helper to pattern code that
// wants to emit additional framework-level progression events (e.g. the
// compensation bracket in the saga pattern).
func PublishFromWorkflow(ctx workflow.Context, pattern, typ string, data any) {
	publishFromWorkflow(ctx, pattern, workflow.GetInfo(ctx), typ, data)
}

// activityInbound publishes step.started / completed / failed around every
// activity execution except PublishEvent itself.
type activityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	root *workerInterceptor
}

func (a *activityInbound) ExecuteActivity(
	ctx context.Context, in *interceptor.ExecuteActivityInput,
) (any, error) {
	info := activity.GetInfo(ctx)
	if info.ActivityType.Name == PublishEventActivityName {
		return a.Next.ExecuteActivity(ctx, in)
	}

	step := info.ActivityType.Name
	attempt := int(info.Attempt)
	start := time.Now()

	a.publishDirect(ctx, info, TypeStepStarted, map[string]any{
		"step":    step,
		"attempt": attempt,
	})

	result, err := a.Next.ExecuteActivity(ctx, in)

	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		a.publishDirect(ctx, info, TypeStepFailed, map[string]any{
			"step":    step,
			"attempt": attempt,
			"error":   err.Error(),
		})
	} else {
		a.publishDirect(ctx, info, TypeStepCompleted, map[string]any{
			"step":       step,
			"attempt":    attempt,
			"durationMs": durationMs,
		})
	}
	return result, err
}

func (a *activityInbound) publishDirect(
	ctx context.Context, info activity.Info, typ string, data any,
) {
	if a.root.publisher == nil {
		return
	}
	env := NewEnvelope(
		a.root.pattern,
		info.WorkflowExecution.ID, info.WorkflowExecution.RunID,
		typ, data,
	)
	if err := a.root.publisher.Publish(ctx, a.root.pattern, env); err != nil {
		activity.GetLogger(ctx).Warn("publish event failed",
			"type", typ, "error", err)
	}
}
