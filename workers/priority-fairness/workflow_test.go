package priorityfairness

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// recordingPublisher captures every business event published by the activities
// so tests can assert on what landed on the wire.
type recordingPublisher struct {
	mu    sync.Mutex
	types []string
}

func (p *recordingPublisher) Publish(_ context.Context, _ string, env events.Envelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.types = append(p.types, env.Type)
	return nil
}

func (*recordingPublisher) Close() error { return nil }

func (p *recordingPublisher) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.types))
	copy(out, p.types)
	return out
}

func TestHelpdeskRunWorkflow_HappyPath(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{Publisher: events.NopPublisher{}})

	env.ExecuteWorkflow(HelpdeskRunWorkflow, HelpdeskInput{FairnessOn: true})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestHelpdeskRunWorkflow_PublishesSeedEvent verifies that the seed phase
// emits exactly one helpdesk.run.seeded envelope before any per-ticket events.
func TestHelpdeskRunWorkflow_PublishesSeedEvent(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pub := &recordingPublisher{}
	env.RegisterActivity(&Activities{Publisher: pub})

	env.ExecuteWorkflow(HelpdeskRunWorkflow, HelpdeskInput{FairnessOn: false})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	published := pub.snapshot()
	require.NotEmpty(t, published)
	require.Equal(t, TypeRunSeeded, published[0],
		"helpdesk.run.seeded must be the first event so the UI can populate queues")

	count := 0
	for _, typ := range published {
		if typ == TypeRunSeeded {
			count++
		}
	}
	require.Equal(t, 1, count, "expected exactly one helpdesk.run.seeded envelope")
}
