package encryption

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Activities groups the encryption pattern activities. Publisher is injected
// so activities stay runnable without NATS (a nil/nop publisher is fine).
type Activities struct {
	Publisher events.Publisher
}

// ValidateOrder checks the order shape and emits an order-validated event.
func (a *Activities) ValidateOrder(ctx context.Context, in SensitiveOrder) error {
	activity.GetLogger(ctx).Info("Validating order", "order", in.OrderID)
	time.Sleep(400 * time.Millisecond)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeOrderValidated, map[string]any{
		"orderId": in.OrderID,
		"items":   len(in.Items),
	})
	return nil
}

// ChargeCard charges the customer's card and returns a payment reference.
func (a *Activities) ChargeCard(ctx context.Context, in SensitiveOrder) (string, error) {
	activity.GetLogger(ctx).Info("Charging card",
		"order", in.OrderID, "amount", in.Total, "last4", in.Customer.CardLast4)
	time.Sleep(600 * time.Millisecond)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeCardCharged, map[string]any{
		"orderId": in.OrderID,
		"amount":  in.Total,
		"last4":   in.Customer.CardLast4,
	})
	return fmt.Sprintf("pay-%s", in.OrderID), nil
}

// ShipOrder dispatches the order and returns a tracking ID.
func (a *Activities) ShipOrder(ctx context.Context, orderID string) (string, error) {
	activity.GetLogger(ctx).Info("Shipping order", "order", orderID)
	time.Sleep(500 * time.Millisecond)
	trackingID := fmt.Sprintf("trk-%s", orderID)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeOrderShipped, map[string]any{
		"orderId":    orderID,
		"trackingId": trackingID,
	})
	return trackingID, nil
}

// SendReceipt emails the customer and emits both receipt-sent and the
// terminal order-completed event.
func (a *Activities) SendReceipt(ctx context.Context, in SensitiveOrder, trackingID string) error {
	activity.GetLogger(ctx).Info("Sending receipt", "email", in.Customer.Email, "tracking", trackingID)
	time.Sleep(300 * time.Millisecond)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeReceiptSent, map[string]any{
		"email":      in.Customer.Email,
		"trackingId": trackingID,
	})
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeOrderCompleted, map[string]any{
		"orderId": in.OrderID,
	})
	return nil
}
