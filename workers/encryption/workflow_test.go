package encryption

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

func TestProcessSensitiveOrderWorkflow_HappyPath(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	a := &Activities{Publisher: events.NopPublisher{}}
	env.RegisterActivityWithOptions(a.ValidateOrder, activity.RegisterOptions{Name: "validate-order"})
	env.RegisterActivityWithOptions(a.ChargeCard, activity.RegisterOptions{Name: "charge-card"})
	env.RegisterActivityWithOptions(a.ShipOrder, activity.RegisterOptions{Name: "ship-order"})
	env.RegisterActivityWithOptions(a.SendReceipt, activity.RegisterOptions{Name: "send-receipt"})

	env.ExecuteWorkflow(ProcessSensitiveOrderWorkflow, SensitiveOrder{
		OrderID: "order-42",
		Customer: Customer{
			Name:      "Alice Example",
			Email:     "alice@example.com",
			CardLast4: "4242",
		},
		Items: []Item{
			{SKU: "SKU-1", Qty: 1, Price: 19.99},
			{SKU: "SKU-2", Qty: 2, Price: 5.00},
		},
		Total: 29.99,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result OrderConfirmation
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "order-42", result.OrderID)
	require.NotEmpty(t, result.PaymentRef)
	require.NotEmpty(t, result.TrackingID)
	require.True(t, result.ReceiptSent)
}
