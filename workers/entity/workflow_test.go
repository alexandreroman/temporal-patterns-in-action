package entity

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// recordingPublisher captures every business event published by the activities
// so tests can assert on the emitted event stream.
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

// registerEntityActivities registers each activity under its kebab-case name,
// mirroring the real worker entrypoint so activity dispatch matches production.
func registerEntityActivities(env *testsuite.TestWorkflowEnvironment, a *Activities) {
	env.RegisterActivityWithOptions(a.ValidateItem, activity.RegisterOptions{Name: "validate-item"})
	env.RegisterActivityWithOptions(a.PriceItem, activity.RegisterOptions{Name: "price-item"})
	env.RegisterActivityWithOptions(a.UpdateQty, activity.RegisterOptions{Name: "update-qty"})
	env.RegisterActivityWithOptions(a.RemoveItem, activity.RegisterOptions{Name: "remove-item"})
	env.RegisterActivityWithOptions(a.ProcessPayment, activity.RegisterOptions{Name: "process-payment"})
	env.RegisterActivityWithOptions(a.SendConfirmation, activity.RegisterOptions{Name: "send-confirmation"})
}

func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}

func TestShoppingCartWorkflow_EmptyCheckout(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pub := &recordingPublisher{}
	registerEntityActivities(env, &Activities{Publisher: pub, FastMode: true})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCheckout, CheckoutSignal{})
	}, time.Second)

	env.ExecuteWorkflow(ShoppingCartWorkflow, CartState{CartID: "cart-empty"})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	published := pub.snapshot()
	require.Contains(t, published, TypePaymentProcessed)
	require.Contains(t, published, TypeConfirmationSent)
	require.NotContains(t, published, TypeItemAdded)
}

func TestShoppingCartWorkflow_AddUpdateRemoveCheckout(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pub := &recordingPublisher{}
	registerEntityActivities(env, &Activities{Publisher: pub, FastMode: true})

	// Sequence signals in simulated time; the test env fast-forwards past
	// sleeps so each callback lands while the workflow is blocked on the
	// signal selector.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddItem, AddItemSignal{
			ItemID: "sku-1", Name: "Keyboard", PriceCents: 4500, Qty: 1,
		})
	}, time.Second)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddItem, AddItemSignal{
			ItemID: "sku-2", Name: "Mouse", PriceCents: 2500, Qty: 2,
		})
	}, 2*time.Second)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalUpdateQty, UpdateQtySignal{ItemID: "sku-1", Qty: 3})
	}, 3*time.Second)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalRemoveItem, RemoveItemSignal{ItemID: "sku-2"})
	}, 4*time.Second)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCheckout, CheckoutSignal{})
	}, 5*time.Second)

	env.ExecuteWorkflow(ShoppingCartWorkflow, CartState{CartID: "cart-happy"})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	published := pub.snapshot()

	// All business events must appear in signal-application order.
	added1 := indexOf(published, TypeItemAdded)
	require.GreaterOrEqual(t, added1, 0, "first item.added must be published")
	updated := indexOf(published, TypeQtyUpdated)
	removed := indexOf(published, TypeItemRemoved)
	paid := indexOf(published, TypePaymentProcessed)
	confirmed := indexOf(published, TypeConfirmationSent)
	require.Less(t, added1, updated, "qty.updated must follow item.added")
	require.Less(t, updated, removed, "item.removed must follow qty.updated")
	require.Less(t, removed, paid, "payment.processed must follow item.removed")
	require.Less(t, paid, confirmed, "confirmation.sent must follow payment.processed")

	// Two items were added (sku-1 and sku-2).
	addedCount := 0
	for _, ev := range published {
		if ev == TypeItemAdded {
			addedCount++
		}
	}
	require.Equal(t, 2, addedCount, "expected exactly two item.added events")
}

func TestShoppingCartWorkflow_GetCartQuery(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	registerEntityActivities(env, &Activities{Publisher: events.NopPublisher{}, FastMode: true})

	var snapshot Progress
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddItem, AddItemSignal{
			ItemID: "sku-1", Name: "Pen", PriceCents: 300, Qty: 2,
		})
	}, time.Second)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddItem, AddItemSignal{
			ItemID: "sku-2", Name: "Pad", PriceCents: 500, Qty: 1,
		})
	}, 2*time.Second)
	env.RegisterDelayedCallback(func() {
		val, err := env.QueryWorkflow(QueryGetCart)
		require.NoError(t, err)
		require.NoError(t, val.Get(&snapshot))
	}, 3*time.Second)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCheckout, CheckoutSignal{})
	}, 4*time.Second)

	env.ExecuteWorkflow(ShoppingCartWorkflow, CartState{CartID: "cart-query"})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	require.Len(t, snapshot.Items, 2)
	require.Equal(t, 2*300+1*500, snapshot.TotalCents)
	require.Equal(t, 1, snapshot.QueriesAnswered)
}
