package events

import (
	"time"

	"github.com/google/uuid"
)

// Event category constants. Category is derived from Envelope.Type and never
// stored as a separate field: types prefixed with "progress." map to
// CategoryProgress; everything else (pattern-prefixed business events like
// "saga.inventory.reserved") maps to CategoryBusiness.
const (
	CategoryProgress = "progress"
	CategoryBusiness = "business"
)

// Progress event types emitted by the framework interceptor for every pattern.
const (
	TypeStepStarted   = "progress.step.started"
	TypeStepCompleted = "progress.step.completed"
	TypeStepFailed    = "progress.step.failed"
)

const (
	specVersion = "1.0"
	timeFormat  = "2006-01-02T15:04:05.000Z07:00"
)

// Envelope is the JSON wire format published on NATS.
type Envelope struct {
	SpecVersion string `json:"specversion"`
	ID          string `json:"id"`
	Source      string `json:"source"`
	Type        string `json:"type"`
	WorkflowID  string `json:"workflowId"`
	RunID       string `json:"runId"`
	Time        string `json:"time"`
	Data        any    `json:"data"`
}

// NewEnvelope builds a fully-populated Envelope with a fresh UUID and
// wall-clock timestamp.
func NewEnvelope(pattern, workflowID, runID, typ string, data any) Envelope {
	return Envelope{
		SpecVersion: specVersion,
		ID:          uuid.NewString(),
		Source:      "patterns." + pattern,
		Type:        typ,
		WorkflowID:  workflowID,
		RunID:       runID,
		Time:        time.Now().UTC().Format(timeFormat),
		Data:        data,
	}
}
