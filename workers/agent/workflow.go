// Package agent implements the durable AI agent pattern: a workflow that
// drives a Reason-Act loop over a scripted LLM and a catalogue of MCP tools,
// with built-in support for retries on transient LLM failures and for a
// human-in-the-loop approval checkpoint delivered as a signal.
package agent

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// maxIterations caps the agent loop so a misbehaving LLM cannot spin forever.
// The scripted demos all terminate well within this bound.
const maxIterations = 12

// PhaseIdle is the initial phase reported by the getProgress query before the
// first LLM call completes. The other phase strings are purely informational
// and documented on Progress.Phase.
const (
	PhaseIdle      = "idle"
	PhaseLLM       = "llm"
	PhaseTool      = "tool"
	PhaseApproval  = "awaiting-approval"
	PhaseCompleted = "completed"
)

// TravelAgentWorkflow runs the agent loop: every iteration calls the LLM with
// the full conversation history, either dispatches an MCP tool, blocks on a
// human approval signal, or returns the final plan. Every activity is a
// natural durability boundary — a crash mid-loop resumes exactly where it
// left off, which is the whole point of the pattern.
func TravelAgentWorkflow(ctx workflow.Context, req UserRequest) (Plan, error) {
	logger := workflow.GetLogger(ctx)

	history := []Message{{Role: RoleUser, Content: req.Prompt}}
	progress := Progress{Phase: PhaseIdle}

	if err := workflow.SetQueryHandler(ctx, "getProgress", func() (Progress, error) {
		return progress, nil
	}); err != nil {
		return Plan{}, err
	}

	// LLM calls use bounded retries so the "retry" scenario can inject one
	// transient timeout per loop and still reach a successful attempt.
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    5,
		},
	})

	// RecordApproval and ReportPlan are pure event-publish side effects — a NATS
	// hiccup must not add ~10s of retry latency to the user-visible approval path.
	fastCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})

	approvalCh := workflow.GetSignalChannel(ctx, SignalApproval)
	var a *Activities

	for i := 1; i <= maxIterations; i++ {
		progress.Loop = i
		progress.Phase = PhaseLLM

		var resp LLMResponse
		if err := workflow.ExecuteActivity(ctx, a.CallLLM, LLMRequest{
			Scenario: req.Scenario,
			Loop:     i,
			History:  history,
		}).Get(ctx, &resp); err != nil {
			return Plan{}, err
		}
		progress.LLMCalls++
		progress.Tokens += resp.Tokens
		history = append(history, resp.Message)

		switch {
		case resp.ToolCall != nil:
			progress.Phase = PhaseTool
			var result ToolResult
			if err := workflow.ExecuteActivity(ctx, a.ExecuteMCPTool, *resp.ToolCall).Get(ctx, &result); err != nil {
				return Plan{}, err
			}
			progress.ToolCalls++
			history = append(history, Message{
				Role:     RoleTool,
				Content:  result.Output,
				ToolName: result.Name,
			})

		case resp.NeedsApproval:
			progress.Phase = PhaseApproval
			var decision ApprovalDecision
			approvalCh.Receive(ctx, &decision)
			if err := workflow.ExecuteActivity(fastCtx, a.RecordApproval, decision).Get(ctx, nil); err != nil {
				logger.Warn("record-approval failed", "error", err)
			}
			if !decision.Approved {
				return Plan{}, temporal.NewNonRetryableApplicationError(
					"user rejected the plan", "AgentRejected", nil)
			}
			history = append(history, Message{
				Role:    RoleSystem,
				Content: "User approved the travel plan.",
			})

		case resp.Plan != nil:
			progress.Phase = PhaseCompleted
			progress.Completed = true
			if err := workflow.ExecuteActivity(fastCtx, a.ReportPlan, *resp.Plan).Get(ctx, nil); err != nil {
				logger.Warn("report-plan failed", "error", err)
			}
			return *resp.Plan, nil
		}
	}
	return Plan{}, fmt.Errorf("agent exceeded %d iterations without a plan", maxIterations)
}
