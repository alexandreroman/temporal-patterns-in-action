package events

import "strings"

// Subject builds the NATS subject for an event:
// patterns.<pattern>.<workflowId>.<category>.
func Subject(pattern, workflowID, category string) string {
	return "patterns." + pattern + "." + workflowID + "." + category
}

// CategoryOf returns the first segment of an event type (e.g. "progress" for
// "progress.workflow.started"). Returns "" when the type has no separator.
func CategoryOf(eventType string) string {
	if i := strings.IndexByte(eventType, '.'); i >= 0 {
		return eventType[:i]
	}
	return ""
}
