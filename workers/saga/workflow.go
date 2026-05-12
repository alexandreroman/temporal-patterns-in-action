// Package saga implements the saga pattern: a distributed transaction with
// compensations that roll back previously-completed steps on failure.
package saga

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// OrderProcessingWorkflow runs a fraud check, prepares the shipment, charges
// the customer, and sends a confirmation. If any step fails, previously-
// completed steps are compensated in reverse order. Compensations run on a
// disconnected context so they execute even if the workflow itself is
// cancelled.
func OrderProcessingWorkflow(ctx workflow.Context, input OrderInput) (OrderResult, error) {
	// txID is the idempotency key every saga activity receives as its first
	// business argument, so a retried attempt can be recognised as the same
	// logical operation by the downstream service.
	txID := input.TransactionID

	// Activities use Temporal's default retry policy — unlimited attempts with
	// exponential backoff — so random activity timeouts are eventually retried
	// to success.
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 6 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	logger := workflow.GetLogger(ctx)

	result := OrderResult{
		OrderID: input.OrderID,
		Status:  "pending",
	}

	var a *Activities
	var compensations []func(workflow.Context) error

	runCompensations := func() {
		disconnected, _ := workflow.NewDisconnectedContext(ctx)
		compCtx := workflow.WithActivityOptions(disconnected, workflow.ActivityOptions{
			StartToCloseTimeout: 6 * time.Second,
		})
		for i := len(compensations) - 1; i >= 0; i-- {
			if err := compensations[i](compCtx); err != nil {
				logger.Error("compensation failed", "error", err)
			}
		}
	}

	// Step 1 — check fraud
	var checkID string
	if err := workflow.ExecuteActivity(ctx, a.CheckFraud, txID, input).Get(ctx, &checkID); err != nil {
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	compensations = append(compensations, func(c workflow.Context) error {
		return workflow.ExecuteActivity(c, a.ReleaseFraudHold, txID, checkID).Get(c, nil)
	})
	result.Confirmed = append(result.Confirmed, checkID)

	// Step 2 — prepare shipment
	var shipmentID string
	if err := workflow.ExecuteActivity(ctx, a.PrepareShipment, txID, input, checkID).Get(ctx, &shipmentID); err != nil {
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	compensations = append(compensations, func(c workflow.Context) error {
		return workflow.ExecuteActivity(c, a.CancelShipment, txID, shipmentID).Get(c, nil)
	})
	result.Confirmed = append(result.Confirmed, shipmentID)

	// Step 3 — charge customer
	var paymentID string
	if err := workflow.ExecuteActivity(ctx, a.ChargeCustomer, txID, input, shipmentID).Get(ctx, &paymentID); err != nil {
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	compensations = append(compensations, func(c workflow.Context) error {
		return workflow.ExecuteActivity(c, a.RefundCustomer, txID, paymentID, input.Amount).Get(c, nil)
	})
	result.Confirmed = append(result.Confirmed, paymentID)

	// Step 4 — send confirmation
	var email string
	if err := workflow.ExecuteActivity(ctx, a.SendConfirmation, txID, input).Get(ctx, &email); err != nil {
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	result.Confirmed = append(result.Confirmed, email)
	result.Status = "completed"
	return result, nil
}
