package events

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
)

// NewInterceptor returns a WorkerInterceptor that publishes framework-level
// progression events around every activity execution. Events are published
// directly to NATS from activity context; workflow code stays free of any
// event-emission side effects.
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

// activityInbound publishes step.started / completed / failed around every
// activity execution.
type activityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	root *workerInterceptor
}

func (a *activityInbound) ExecuteActivity(
	ctx context.Context, in *interceptor.ExecuteActivityInput,
) (any, error) {
	info := activity.GetInfo(ctx)
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
