// Package encryption implements the payload encryption demo: client and
// worker share an AES-256-GCM codec so Temporal stores only ciphertext
// while the workflow and activities see cleartext.
package encryption

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// ProcessSensitiveOrderWorkflow runs a linear validate → charge → ship →
// receipt pipeline. This is not a saga — there are no compensations.
func ProcessSensitiveOrderWorkflow(ctx workflow.Context, in SensitiveOrder) (OrderConfirmation, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 6 * time.Second,
	})

	var a *Activities

	if err := workflow.ExecuteActivity(ctx, a.ValidateOrder, in).Get(ctx, nil); err != nil {
		return OrderConfirmation{}, err
	}

	var paymentRef string
	if err := workflow.ExecuteActivity(ctx, a.ChargeCard, in).Get(ctx, &paymentRef); err != nil {
		return OrderConfirmation{}, err
	}

	var trackingID string
	if err := workflow.ExecuteActivity(ctx, a.ShipOrder, in.OrderID).Get(ctx, &trackingID); err != nil {
		return OrderConfirmation{}, err
	}

	if err := workflow.ExecuteActivity(ctx, a.SendReceipt, in, trackingID).Get(ctx, nil); err != nil {
		return OrderConfirmation{}, err
	}

	return OrderConfirmation{
		OrderID:     in.OrderID,
		Status:      "completed",
		PaymentRef:  paymentRef,
		TrackingID:  trackingID,
		ReceiptSent: true,
	}, nil
}
