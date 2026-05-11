package priorityfairness

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
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

// stubStartResolveTicket mocks the StartResolveTicket local activity so the
// test does not need a live Temporal client. For every dispatched ticket the
// mock immediately schedules a SignalTicketDone callback on the test env,
// driving the parent's drain loop without actually running per-ticket
// workflows.
func stubStartResolveTicket(env *testsuite.TestWorkflowEnvironment, a *Activities) {
	env.OnActivity(a.StartResolveTicket, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in StartResolveTicketInput) error {
			env.RegisterDelayedCallback(func() {
				env.SignalWorkflow(SignalTicketDone, in.Ticket.ID)
			}, time.Millisecond)
			return nil
		},
	)
}

func TestHelpdeskRunWorkflow_HappyPath(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	a := &Activities{Publisher: events.NopPublisher{}}
	env.RegisterActivity(a)
	stubStartResolveTicket(env, a)

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
	a := &Activities{Publisher: pub}
	env.RegisterActivity(a)
	stubStartResolveTicket(env, a)

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
