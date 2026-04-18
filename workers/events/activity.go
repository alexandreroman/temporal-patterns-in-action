package events

import (
	"context"
	"strings"
)

// PublishEventActivityName is the registered name of the publishing activity.
// Workflows call it via workflow.ExecuteLocalActivity(ctx, PublishEventActivityName, env).
const PublishEventActivityName = "PublishEvent"

// Activity bundles the shared publisher so it can be injected as dependency
// for the PublishEvent activity. Register the struct directly — its
// PublishEvent method is picked up under its own name:
//
//	w.RegisterActivity(&events.Activity{Publisher: pub})
type Activity struct {
	Publisher Publisher
}

// PublishEvent normalizes the envelope, resolves the pattern from env.Source
// and publishes via the injected Publisher. Safe to run from a local activity
// invoked inside a workflow.
func (a *Activity) PublishEvent(ctx context.Context, env Envelope) error {
	if a == nil || a.Publisher == nil {
		return nil
	}
	env.normalize()
	pattern := strings.TrimPrefix(env.Source, "patterns.")
	return a.Publisher.Publish(ctx, pattern, env)
}
