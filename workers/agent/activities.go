package agent

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Animation delays tuned so the UI has time to render each state transition
// without making the demo feel sluggish.
const (
	llmThinkTime = 1500 * time.Millisecond
	toolWorkTime = 1000 * time.Millisecond
)

// Activities groups the agent pattern activities. Fields can be used for
// dependency injection (LLM client, MCP client, event publisher, ...). The
// demo implementation is fully scripted to keep every run reproducible.
type Activities struct {
	Publisher events.Publisher
	// FastMode skips the simulated think/work sleeps so tests finish quickly.
	FastMode bool
}

func (a *Activities) pause(d time.Duration) {
	if a.FastMode {
		return
	}
	time.Sleep(d)
}

// CallLLM simulates a round-trip to an LLM. It publishes an agent.llm.responded
// business event carrying the decision the LLM made (a tool call, an approval
// request, or the final plan). The retry scenario hangs the first attempt of
// loop 3 past the start-to-close timeout so Temporal's retry policy carries
// the activity to a successful second attempt.
func (a *Activities) CallLLM(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	info := activity.GetInfo(ctx)
	attempt := int(info.Attempt)
	activity.GetLogger(ctx).Info("calling LLM",
		"scenario", req.Scenario, "loop", req.Loop, "attempt", attempt)

	// Re-publishing the user prompt on every retry is fine — event consumers
	// dedupe by envelope id, not by type.
	if req.Loop == 1 && attempt == 1 {
		events.PublishBusiness(ctx, a.Publisher, Pattern, TypeUserPrompt, map[string]any{
			"prompt": firstUserMessage(req.History),
		})
	}

	if req.Scenario == ScenarioRetry && req.Loop == 3 && attempt == 1 {
		activity.GetLogger(ctx).Info("injecting LLM timeout for retry demo")
		<-ctx.Done()
		return LLMResponse{}, ctx.Err()
	}

	a.pause(llmThinkTime)

	resp := scriptedLLM(req)

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeLLMResponded, map[string]any{
		"loop":          req.Loop,
		"attempt":       attempt,
		"message":       resp.Message,
		"toolCall":      resp.ToolCall,
		"needsApproval": resp.NeedsApproval,
		"plan":          resp.Plan,
		"tokens":        resp.Tokens,
	})

	if resp.NeedsApproval {
		events.PublishBusiness(ctx, a.Publisher, Pattern, TypeApprovalRequested, map[string]any{
			"loop":    req.Loop,
			"summary": resp.Message.Content,
		})
	}

	return resp, nil
}

// ExecuteMCPTool simulates running an MCP tool. Tool results are scripted so
// the conversation the LLM sees on the next loop stays deterministic.
func (a *Activities) ExecuteMCPTool(ctx context.Context, call ToolCall) (ToolResult, error) {
	activity.GetLogger(ctx).Info("executing MCP tool", "name", call.Name)

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeToolStarted, map[string]any{
		"name": call.Name,
		"args": call.Args,
	})

	a.pause(toolWorkTime)

	output := scriptedToolOutput(call.Name)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeToolCompleted, map[string]any{
		"name":   call.Name,
		"output": output,
	})
	return ToolResult{Name: call.Name, Output: output}, nil
}

// RecordApproval publishes the approval decision as a business event. The
// workflow is only interested in whether to continue, but the UI needs to see
// an explicit "resumed" marker — this activity carries that signal.
func (a *Activities) RecordApproval(ctx context.Context, decision ApprovalDecision) error {
	activity.GetLogger(ctx).Info("recording approval", "approved", decision.Approved)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeApprovalReceived, map[string]any{
		"approved": decision.Approved,
	})
	return nil
}

// ReportPlan publishes the final plan so the UI can display it alongside the
// conversation without having to query the workflow result.
func (a *Activities) ReportPlan(ctx context.Context, plan Plan) error {
	activity.GetLogger(ctx).Info("reporting plan", "summary", plan.Summary)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypePlanReady, map[string]any{
		"summary": plan.Summary,
	})
	return nil
}

func firstUserMessage(history []Message) string {
	for _, m := range history {
		if m.Role == RoleUser {
			return m.Content
		}
	}
	return ""
}

// scriptedTokens returns a realistic-looking (non-round) token count per loop,
// growing as conversation history accumulates.
func scriptedTokens(loop int) int {
	table := []int{742, 918, 1187, 1463, 1724}
	if loop >= 1 && loop <= len(table) {
		return table[loop-1]
	}
	return table[len(table)-1]
}

// scriptedLLM returns the agent's next decision for a given scenario/loop.
// Keeping every trajectory in a single switch makes it easy to read the
// whole demo at a glance, and fully determines what the UI will show.
func scriptedLLM(req LLMRequest) LLMResponse {
	tokens := scriptedTokens(req.Loop)
	switch req.Loop {
	case 1:
		return LLMResponse{
			Message: Message{
				Role:     RoleLLM,
				Content:  "I'll search for direct flights to Tokyo in October.",
				ToolName: "search_flights",
			},
			ToolCall: &ToolCall{Name: "search_flights", Args: map[string]any{
				"destination": "Tokyo", "month": "October",
			}},
			Tokens: tokens,
		}
	case 2:
		return LLMResponse{
			Message: Message{
				Role:     RoleLLM,
				Content:  "Good options found. Searching hotels near Shinjuku.",
				ToolName: "search_hotels",
			},
			ToolCall: &ToolCall{Name: "search_hotels", Args: map[string]any{
				"area": "Shinjuku",
			}},
			Tokens: tokens,
		}
	case 3:
		return LLMResponse{
			Message: Message{
				Role:     RoleLLM,
				Content:  "Let me verify your calendar availability.",
				ToolName: "get_calendar",
			},
			ToolCall: &ToolCall{Name: "get_calendar", Args: map[string]any{
				"month": "October",
			}},
			Tokens: tokens,
		}
	case 4:
		if req.Scenario == ScenarioApproval {
			return LLMResponse{
				Message: Message{
					Role:    RoleLLM,
					Content: "Plan ready: ANA flight $890 + Granbell 5 nights $600 = $1490 total. Requesting approval.",
				},
				NeedsApproval: true,
				Tokens:        tokens,
			}
		}
		return finalPlanResponse(tokens)
	case 5:
		if req.Scenario == ScenarioApproval {
			return LLMResponse{
				Message: Message{
					Role:     RoleLLM,
					Content:  "Approval received — booking the flight now.",
					ToolName: "book_flight",
				},
				ToolCall: &ToolCall{Name: "book_flight", Args: map[string]any{
					"airline": "ANA", "reference": "ANA-4821",
				}},
				Tokens: tokens,
			}
		}
		return finalPlanResponse(tokens)
	}
	return finalPlanResponse(tokens)
}

func finalPlanResponse(tokens int) LLMResponse {
	return LLMResponse{
		Message: Message{
			Role:    RoleLLM,
			Content: "Here's your Tokyo itinerary: Oct 12-17, ANA flight $890, Granbell Hotel 5 nights $600. Day-by-day plan included.",
		},
		Plan:   &Plan{Summary: "Tokyo, Oct 12-17 — ANA flight + Granbell Hotel (5 nights) — $1,490 total"},
		Tokens: tokens,
	}
}

func scriptedToolOutput(name string) string {
	switch name {
	case "search_flights":
		return "3 flights: ANA $890 direct, JAL $920 direct, UA $680 1-stop"
	case "search_hotels":
		return "4 hotels: Shinjuku Granbell $120/n, Shibuya Stream $180/n, Park Hyatt $420/n, Keio Plaza $150/n"
	case "get_calendar":
		return "Oct 10-20 free. Oct 5-9 has meetings."
	case "book_flight":
		return "Booked ANA NH-107 — confirmation ANA-4821"
	case "book_hotel":
		return "Booked Granbell Shinjuku — confirmation GB-7210"
	case "send_itinerary":
		return "Itinerary emailed to traveller"
	}
	return "ok"
}
