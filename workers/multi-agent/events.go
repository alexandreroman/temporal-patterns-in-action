package multiagent

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "multi-agent"

// Multi-agent business event types. Prefixed with the pattern name to
// guarantee no collision with types emitted by other patterns.
const (
	TypeUserPrompt     = "multi-agent.user.prompt"
	TypePlanReady      = "multi-agent.plan.ready"
	TypeQueriesReady   = "multi-agent.queries.ready"
	TypeFanoutStarted  = "multi-agent.fanout.started"
	TypeSearchStarted  = "multi-agent.search.started"
	TypeSearchComplete = "multi-agent.search.completed"
	TypeSearchFailed   = "multi-agent.search.failed"
	TypeChildCompleted = "multi-agent.child.completed"
	TypeChildFailed    = "multi-agent.child.failed"
	TypeReportReady    = "multi-agent.report.ready"
)
