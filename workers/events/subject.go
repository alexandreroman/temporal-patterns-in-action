package events

import "strings"

// Subject builds the NATS subject for an event:
// patterns.<pattern>.<workflowId>.<category>.
func Subject(pattern, workflowID, category string) string {
	return "patterns." + pattern + "." + workflowID + "." + category
}

// CategoryOf classifies an event type into a subject category. Types prefixed
// with "progress." are framework lifecycle events (CategoryProgress); anything
// else is a pattern-specific business event (CategoryBusiness). An empty
// string maps to "" so callers can distinguish a missing type from a valid
// one.
func CategoryOf(eventType string) string {
	if eventType == "" {
		return ""
	}
	if strings.HasPrefix(eventType, "progress.") {
		return CategoryProgress
	}
	return CategoryBusiness
}
