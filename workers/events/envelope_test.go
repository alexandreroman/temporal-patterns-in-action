package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEnvelopePopulatesContract(t *testing.T) {
	env := NewEnvelope("saga", "wf-1", "run-1", TypeWorkflowStarted, struct{}{})

	if env.SpecVersion != "1.0" {
		t.Errorf("specversion = %q, want %q", env.SpecVersion, "1.0")
	}
	if env.ID == "" {
		t.Error("ID must be non-empty")
	}
	if env.Source != "patterns.saga" {
		t.Errorf("Source = %q, want %q", env.Source, "patterns.saga")
	}
	if env.Type != TypeWorkflowStarted {
		t.Errorf("Type = %q, want %q", env.Type, TypeWorkflowStarted)
	}
	if env.WorkflowID != "wf-1" || env.RunID != "run-1" {
		t.Errorf("ids wrong: %q / %q", env.WorkflowID, env.RunID)
	}
	if _, err := time.Parse(timeFormat, env.Time); err != nil {
		t.Errorf("Time %q does not parse: %v", env.Time, err)
	}
}

func TestEnvelopeJSONShape(t *testing.T) {
	env := NewEnvelope("saga", "wf-1", "run-1", TypeStepCompleted, map[string]any{
		"step":       "ReserveCar",
		"attempt":    1,
		"durationMs": 42,
	})

	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	required := []string{"specversion", "id", "source", "type", "workflowId", "runId", "time", "data"}
	for _, k := range required {
		if _, ok := got[k]; !ok {
			t.Errorf("missing field %q in JSON: %s", k, raw)
		}
	}
	if _, ok := got["category"]; ok {
		t.Errorf("unexpected field %q in JSON: %s", "category", raw)
	}
}

func TestWorkflowEnvelopeNormalize(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	env := NewWorkflowEnvelope("saga", "wf-1", "run-1", TypeWorkflowStarted, now, struct{}{})

	if env.ID != "" {
		t.Error("workflow envelope must leave ID empty for activity-side generation")
	}
	if env.Time == "" {
		t.Error("workflow envelope Time must be set from the deterministic clock")
	}

	env.normalize()
	if env.ID == "" {
		t.Error("normalize must fill ID")
	}
	if env.SpecVersion != "1.0" {
		t.Errorf("normalize must fill SpecVersion, got %q", env.SpecVersion)
	}
}
