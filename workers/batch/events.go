package batch

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
