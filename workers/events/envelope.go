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
	TypeWorkflowStarted       = "progress.workflow.started"
	TypeWorkflowCompleted     = "progress.workflow.completed"
	TypeWorkflowFailed        = "progress.workflow.failed"
	TypeStepStarted           = "progress.step.started"
	TypeStepCompleted         = "progress.step.completed"
	TypeStepFailed            = "progress.step.failed"
	TypeCompensationStarted   = "progress.compensation.started"
	TypeCompensationCompleted = "progress.compensation.completed"
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

// NewEnvelope builds a fully-populated Envelope. Use from non-workflow
// contexts only — the fresh UUID and wall-clock time are non-deterministic.
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

// NewWorkflowEnvelope builds an Envelope safe to emit from a workflow: caller
// supplies a deterministic time (e.g. workflow.Now). The ID is left empty and
// filled in by the PublishEvent activity before the event is sent on the wire.
func NewWorkflowEnvelope(pattern, workflowID, runID, typ string, now time.Time, data any) Envelope {
	return Envelope{
		SpecVersion: specVersion,
		Source:      "patterns." + pattern,
		Type:        typ,
		WorkflowID:  workflowID,
		RunID:       runID,
		Time:        now.UTC().Format(timeFormat),
		Data:        data,
	}
}

// normalize fills in any fields that must be generated at publish time.
// Called by the PublishEvent activity on envelopes built from workflow code.
func (e *Envelope) normalize() {
	if e.SpecVersion == "" {
		e.SpecVersion = specVersion
	}
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Time == "" {
		e.Time = time.Now().UTC().Format(timeFormat)
	}
}
