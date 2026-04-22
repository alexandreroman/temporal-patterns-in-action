package entity

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// mainStepDelay is the simulated step delay tuned so the UI has time to render
// each state transition without making the demo feel sluggish.
const mainStepDelay = 600 * time.Millisecond

// Activities groups the entity pattern activities. Fields can be used for
// dependency injection (catalog client, payment gateway, event publisher, ...).
type Activities struct {
	Publisher events.Publisher
	// FastMode skips the simulated step delays so tests finish quickly.
	FastMode bool
}

func (a *Activities) pause(d time.Duration) {
	if a.FastMode {
		return
	}
	time.Sleep(d)
}

// ValidateItem simulates a product-catalog lookup. The step appears in the
// progress.step.* event stream via the activity interceptor; no business
// event is published here.
func (a *Activities) ValidateItem(ctx context.Context, item AddItemSignal) error {
	activity.GetLogger(ctx).Info("validating item",
		"itemId", item.ItemID, "name", item.Name)
	a.pause(mainStepDelay)
	return nil
}

// PriceItem returns the item price. The demo echoes the caller-supplied price
// because the cart is scripted; a real catalog would look it up. The
// entity.item.added business event is emitted here, once the item has a
// definitive price.
func (a *Activities) PriceItem(ctx context.Context, item AddItemSignal) (int, error) {
	activity.GetLogger(ctx).Info("pricing item", "itemId", item.ItemID)
	a.pause(mainStepDelay)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeItemAdded, map[string]any{
		"itemId":     item.ItemID,
		"name":       item.Name,
		"priceCents": item.PriceCents,
		"qty":        item.Qty,
	})
	return item.PriceCents, nil
}

// UpdateQty records a quantity change for an item already in the cart.
func (a *Activities) UpdateQty(ctx context.Context, sig UpdateQtySignal) error {
	activity.GetLogger(ctx).Info("updating qty", "itemId", sig.ItemID, "qty", sig.Qty)
	a.pause(mainStepDelay)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeQtyUpdated, map[string]any{
		"itemId": sig.ItemID,
		"qty":    sig.Qty,
	})
	return nil
}

// RemoveItem drops an item from the cart.
func (a *Activities) RemoveItem(ctx context.Context, sig RemoveItemSignal) error {
	activity.GetLogger(ctx).Info("removing item", "itemId", sig.ItemID)
	a.pause(mainStepDelay)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeItemRemoved, map[string]any{
		"itemId": sig.ItemID,
	})
	return nil
}

// ProcessPayment simulates charging the customer and returns an order ID.
func (a *Activities) ProcessPayment(ctx context.Context, cartID string, totalCents int) (string, error) {
	activity.GetLogger(ctx).Info("processing payment", "cartId", cartID, "amountCents", totalCents)
	a.pause(mainStepDelay)
	orderID := fmt.Sprintf("ord-%s", cartID)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypePaymentProcessed, map[string]any{
		"orderId":     orderID,
		"amountCents": totalCents,
	})
	return orderID, nil
}

// SendConfirmation emails the customer that the order is confirmed.
func (a *Activities) SendConfirmation(ctx context.Context, cartID, orderID string) (string, error) {
	activity.GetLogger(ctx).Info("sending confirmation", "cartId", cartID, "orderId", orderID)
	a.pause(mainStepDelay)
	email := fmt.Sprintf("%s@example.com", cartID)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeConfirmationSent, map[string]any{
		"email":   email,
		"orderId": orderID,
	})
	return email, nil
}
