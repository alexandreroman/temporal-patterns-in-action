package events

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
)

// NewInterceptor returns a WorkerInterceptor that publishes framework-level
// progression events around every regular activity execution. Events are
// published directly to NATS from activity context; workflow code stays free
// of any event-emission side effects.
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
// regular (non-local) activity execution.
type activityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	root *workerInterceptor
}

func (a *activityInbound) ExecuteActivity(
	ctx context.Context, in *interceptor.ExecuteActivityInput,
) (any, error) {
	info := activity.GetInfo(ctx)
	// Local activities are orchestration plumbing, not demo steps.
	if info.IsLocalActivity {
		return a.Next.ExecuteActivity(ctx, in)
	}
	pub, pattern := a.root.publisher, a.root.pattern
	wfID, runID := info.WorkflowExecution.ID, info.WorkflowExecution.RunID
	step := info.ActivityType.Name
	attempt := int(info.Attempt)
	start := time.Now()

	PublishBusinessAs(ctx, pub, pattern, wfID, runID, TypeStepStarted, map[string]any{
		"step":    step,
		"attempt": attempt,
	})

	result, err := a.Next.ExecuteActivity(ctx, in)

	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		PublishBusinessAs(ctx, pub, pattern, wfID, runID, TypeStepFailed, map[string]any{
			"step":    step,
			"attempt": attempt,
			"error":   err.Error(),
		})
	} else {
		PublishBusinessAs(ctx, pub, pattern, wfID, runID, TypeStepCompleted, map[string]any{
			"step":       step,
			"attempt":    attempt,
			"durationMs": durationMs,
		})
	}
	return result, err
}
