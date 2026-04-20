package agent

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "agent"

// Agent-specific business event types. Prefixed with the pattern name to
// guarantee no collision with types emitted by other patterns.
const (
	TypeUserPrompt        = "agent.user.prompt"
	TypeLLMResponded      = "agent.llm.responded"
	TypeToolStarted       = "agent.tool.started"
	TypeToolCompleted     = "agent.tool.completed"
	TypeApprovalRequested = "agent.approval.requested"
	TypeApprovalReceived  = "agent.approval.received"
	TypePlanReady         = "agent.plan.ready"
)
