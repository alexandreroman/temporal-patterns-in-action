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

const mainStepDelay = 2500 * time.Millisecond

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

// CheckFraud screens the order for fraud and returns a fraud-check hold ID
// that downstream compensation can release.
func (a *Activities) CheckFraud(ctx context.Context, txID string, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Checking fraud",
		"customer", input.CustomerID, "order", input.OrderID, "transactionId", txID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "fraud" {
		return "", temporal.NewNonRetryableApplicationError(
			"fraud detected", "FraudDetected", nil)
	}
	checkID := fmt.Sprintf("chk-%s", input.OrderID)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeFraudChecked, map[string]any{"checkId": checkID})
	return checkID, nil
}

// ReleaseFraudHold compensates CheckFraud.
func (a *Activities) ReleaseFraudHold(ctx context.Context, txID string, checkID string) error {
	activity.GetLogger(ctx).Info("Releasing fraud hold", "id", checkID, "transactionId", txID)
	time.Sleep(mainStepDelay)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeFraudReleased, map[string]any{"checkId": checkID})
	return nil
}

// PrepareShipment reserves a shipment slot for the order and returns a
// shipment ID that its compensation can cancel.
func (a *Activities) PrepareShipment(ctx context.Context, txID string, input OrderInput, checkID string) (string, error) {
	activity.GetLogger(ctx).Info("Preparing shipment",
		"order", input.OrderID, "check", checkID, "transactionId", txID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "shipment" {
		return "", temporal.NewNonRetryableApplicationError(
			"shipment unavailable", "ShipmentUnavailable", nil)
	}
	shipmentID := fmt.Sprintf("shp-%s", input.OrderID)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeShipmentPrepared, map[string]any{"shipmentId": shipmentID})
	return shipmentID, nil
}

// CancelShipment compensates PrepareShipment.
func (a *Activities) CancelShipment(ctx context.Context, txID string, shipmentID string) error {
	activity.GetLogger(ctx).Info("Cancelling shipment", "shipment", shipmentID, "transactionId", txID)
	time.Sleep(mainStepDelay)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeShipmentCancelled, map[string]any{"shipmentId": shipmentID})
	return nil
}

// ChargeCustomer charges the customer for the order. The shipment ID keeps
// the call idempotent on the payment provider side.
func (a *Activities) ChargeCustomer(ctx context.Context, txID string, input OrderInput, shipmentID string) (string, error) {
	activity.GetLogger(ctx).Info("Charging customer",
		"customer", input.CustomerID, "amount", input.Amount, "shipment", shipmentID, "transactionId", txID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "charge" {
		return "", temporal.NewNonRetryableApplicationError(
			"payment declined", "PaymentDeclined", nil)
	}
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeCustomerCharged, map[string]any{"amount": input.Amount})
	return fmt.Sprintf("pay-%s", shipmentID), nil
}

// RefundCustomer compensates ChargeCustomer.
func (a *Activities) RefundCustomer(ctx context.Context, txID string, paymentID string, amount int) error {
	activity.GetLogger(ctx).Info("Refunding customer", "payment", paymentID, "amount", amount, "transactionId", txID)
	time.Sleep(mainStepDelay)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeCustomerRefunded, map[string]any{"amount": amount})
	return nil
}

// SendConfirmation emails the customer that the order is confirmed.
func (a *Activities) SendConfirmation(ctx context.Context, txID string, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Sending confirmation", "customer", input.CustomerID, "transactionId", txID)
	if err := a.maybeInjectRandomTimeout(ctx); err != nil {
		return "", err
	}
	time.Sleep(mainStepDelay)
	if input.FailAt == "notification" {
		return "", temporal.NewNonRetryableApplicationError(
			"notification unavailable", "NotificationUnavailable", nil)
	}
	email := fmt.Sprintf("%s@example.com", input.CustomerID)
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeConfirmationSent, map[string]any{"email": email})
	return email, nil
}
