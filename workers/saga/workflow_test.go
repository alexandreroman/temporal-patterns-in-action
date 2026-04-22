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

// recordingPublisher captures every business event published by the activities
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

func (*recordingPublisher) Close() error { return nil }

func (p *recordingPublisher) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.types))
	copy(out, p.types)
	return out
}

func registerTestActivities(env *testsuite.TestWorkflowEnvironment, pub events.Publisher) {
	env.RegisterActivity(&Activities{Publisher: pub})
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

func TestOrderProcessingWorkflow_ChargeFails(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerTestActivities(env, events.NopPublisher{})

	env.ExecuteWorkflow(OrderProcessingWorkflow, OrderInput{
		CustomerID: "bob",
		OrderID:    "order-456",
		Amount:     2000,
		FailAt:     "charge",
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "PaymentDeclined", appErr.Type())
}

// TestOrderProcessingWorkflow_NotificationFails verifies that when the final
// step fails, every prior compensation runs in reverse order.
func TestOrderProcessingWorkflow_NotificationFails(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pub := &recordingPublisher{}
	registerTestActivities(env, pub)

	env.ExecuteWorkflow(OrderProcessingWorkflow, OrderInput{
		CustomerID: "carol",
		OrderID:    "order-789",
		Amount:     1500,
		FailAt:     "notification",
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "NotificationUnavailable", appErr.Type())

	published := pub.snapshot()
	require.Contains(t, published, TypeFraudChecked)
	require.Contains(t, published, TypeShipmentPrepared)
	require.Contains(t, published, TypeCustomerCharged)
	require.Contains(t, published, TypeCustomerRefunded)
	require.Contains(t, published, TypeShipmentCancelled)
	require.Contains(t, published, TypeFraudReleased)

	// Reverse order: refund → cancel shipment → release fraud hold.
	refundIdx := indexOf(published, TypeCustomerRefunded)
	cancelIdx := indexOf(published, TypeShipmentCancelled)
	releaseIdx := indexOf(published, TypeFraudReleased)
	require.Less(t, refundIdx, cancelIdx, "refund must run before shipment cancellation")
	require.Less(t, cancelIdx, releaseIdx, "shipment cancellation must run before fraud hold release")
}

func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}
