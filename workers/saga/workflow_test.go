package saga

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// recordingPublisher captures every domain event published by the activities
// so tests can assert on compensation execution.
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

func (p *recordingPublisher) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.types))
	copy(out, p.types)
	return out
}

// registerTestActivities wires the saga activities plus the shared PublishEvent
// local activity so the framework interceptor has a no-op target for
// progression events.
func registerTestActivities(env *testsuite.TestWorkflowEnvironment, pub events.Publisher) {
	env.RegisterActivity(&Activities{Publisher: pub})
	env.RegisterActivity(&events.Activity{Publisher: events.NopPublisher{}})
}

func TestOrderProcessingWorkflow_HappyPath(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerTestActivities(env, events.NopPublisher{})

	env.ExecuteWorkflow(OrderProcessingWorkflow, OrderInput{
		CustomerID: "alice",
		OrderID:    "order-123",
		Amount:     1200,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result OrderResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "completed", result.Status)
	require.Len(t, result.Confirmed, 4)
}

func TestOrderProcessingWorkflow_PaymentFails(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerTestActivities(env, events.NopPublisher{})

	env.ExecuteWorkflow(OrderProcessingWorkflow, OrderInput{
		CustomerID: "bob",
		OrderID:    "order-456",
		Amount:     2000,
		FailAt:     "payment",
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "PaymentDeclined", appErr.Type())
}

// TestOrderProcessingWorkflow_ShippingFails verifies that when shipping fails
// the prior compensations (refund payment, release inventory) both run in
// reverse order.
func TestOrderProcessingWorkflow_ShippingFails(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pub := &recordingPublisher{}
	registerTestActivities(env, pub)

	env.ExecuteWorkflow(OrderProcessingWorkflow, OrderInput{
		CustomerID: "carol",
		OrderID:    "order-789",
		Amount:     1500,
		FailAt:     "shipping",
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "ShippingUnavailable", appErr.Type())

	published := pub.snapshot()
	require.Contains(t, published, DomainInventoryReserved)
	require.Contains(t, published, DomainPaymentCharged)
	require.Contains(t, published, DomainPaymentRefunded)
	require.Contains(t, published, DomainInventoryReleased)

	// Reverse order: refund happens before release.
	refundIdx := indexOf(published, DomainPaymentRefunded)
	releaseIdx := indexOf(published, DomainInventoryReleased)
	require.Less(t, refundIdx, releaseIdx, "refund must run before inventory release")
}

func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}
