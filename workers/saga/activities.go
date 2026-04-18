package saga

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

const (
	mainStepDelay     = 2500 * time.Millisecond
	compensationDelay = mainStepDelay
)

// Activities groups the saga pattern activities. Fields can be used for
// dependency injection (HTTP clients, DB handles, event publisher, etc.).
type Activities struct {
	Publisher events.Publisher
	// TimeoutChance is the probability per attempt that a main activity hangs
	// past StartToCloseTimeout so Temporal must retry it. Zero disables
	// injection (default for tests).
	TimeoutChance float64
}

// maybeInjectRandomTimeout hangs the activity past StartToCloseTimeout with a
// random chance on every attempt, so Temporal retries it. The workflow uses
// the default unlimited retry policy, so the activity eventually succeeds.
func (a *Activities) maybeInjectRandomTimeout(ctx context.Context) error {
	if a.TimeoutChance <= 0 || rand.Float64() >= a.TimeoutChance {
		return nil
	}
	activity.GetLogger(ctx).Info("Injecting random activity timeout")
	<-ctx.Done()
	return ctx.Err()
}

// ReserveInventory reserves stock for the order and returns an item/reservation ID.
func (a *Activities) ReserveInventory(ctx context.Context, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Reserving inventory",
		"customer", input.CustomerID, "order", input.OrderID, "transactionId", input.TransactionID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "inventory" {
		return "", temporal.NewNonRetryableApplicationError(
			"inventory unavailable", "InventoryUnavailable", nil)
	}
	itemID := fmt.Sprintf("inv-%s", input.OrderID)
	a.publishBusiness(ctx, TypeInventoryReserved, map[string]any{"itemId": itemID})
	return itemID, nil
}

// ReleaseInventory compensates ReserveInventory.
func (a *Activities) ReleaseInventory(ctx context.Context, itemID string) error {
	activity.GetLogger(ctx).Info("Releasing inventory", "id", itemID)
	time.Sleep(compensationDelay)
	a.publishBusiness(ctx, TypeInventoryReleased, map[string]any{"itemId": itemID})
	return nil
}

// ChargePayment charges the customer for the order. The reservation ID keeps
// the call idempotent on the payment provider side.
func (a *Activities) ChargePayment(ctx context.Context, input OrderInput, reservationID string) (string, error) {
	activity.GetLogger(ctx).Info("Charging payment",
		"customer", input.CustomerID, "amount", input.Amount, "reservation", reservationID, "transactionId", input.TransactionID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "payment" {
		return "", temporal.NewNonRetryableApplicationError(
			"payment declined", "PaymentDeclined", nil)
	}
	a.publishBusiness(ctx, TypePaymentCharged, map[string]any{"amount": input.Amount})
	return fmt.Sprintf("pay-%s", reservationID), nil
}

// RefundPayment compensates ChargePayment.
func (a *Activities) RefundPayment(ctx context.Context, paymentID string, amount int) error {
	activity.GetLogger(ctx).Info("Refunding payment", "payment", paymentID, "amount", amount)
	time.Sleep(compensationDelay)
	a.publishBusiness(ctx, TypePaymentRefunded, map[string]any{"amount": amount})
	return nil
}

// ShipOrder dispatches the order and returns a tracking ID.
func (a *Activities) ShipOrder(ctx context.Context, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Shipping order", "order", input.OrderID, "transactionId", input.TransactionID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "shipping" {
		return "", temporal.NewNonRetryableApplicationError(
			"shipping unavailable", "ShippingUnavailable", nil)
	}
	trackingID := fmt.Sprintf("trk-%s", input.OrderID)
	a.publishBusiness(ctx, TypeOrderShipped, map[string]any{"trackingId": trackingID})
	return trackingID, nil
}

// CancelShipment compensates ShipOrder.
func (a *Activities) CancelShipment(ctx context.Context, trackingID string) error {
	activity.GetLogger(ctx).Info("Cancelling shipment", "tracking", trackingID)
	time.Sleep(compensationDelay)
	a.publishBusiness(ctx, TypeShipmentCancelled, map[string]any{"trackingId": trackingID})
	return nil
}

// SendConfirmation emails the customer that the order is confirmed.
func (a *Activities) SendConfirmation(ctx context.Context, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Sending confirmation", "customer", input.CustomerID, "transactionId", input.TransactionID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "notification" {
		return "", temporal.NewNonRetryableApplicationError(
			"notification unavailable", "NotificationUnavailable", nil)
	}
	email := fmt.Sprintf("%s@example.com", input.CustomerID)
	a.publishBusiness(ctx, TypeConfirmationSent, map[string]any{"email": email})
	return email, nil
}

// RetractEmail compensates SendConfirmation.
func (a *Activities) RetractEmail(ctx context.Context, email string) error {
	activity.GetLogger(ctx).Info("Retracting confirmation email", "email", email)
	time.Sleep(compensationDelay)
	a.publishBusiness(ctx, TypeEmailRetracted, map[string]any{"email": email})
	return nil
}
