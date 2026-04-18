// Package saga implements the saga pattern: a distributed transaction with
// compensations that roll back previously-completed steps on failure.
package saga

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// OrderProcessingWorkflow reserves inventory, charges the customer, ships the
// order, and sends a confirmation email. If any step fails, previously-
// completed steps are compensated in reverse order. Compensations run on a
// disconnected context so they execute even if the workflow itself is
// cancelled.
func OrderProcessingWorkflow(ctx workflow.Context, input OrderInput) (OrderResult, error) {
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
	progress := Progress{CurrentStep: "starting"}

	if err := workflow.SetQueryHandler(ctx, "getProgress", func() (Progress, error) {
		return progress, nil
	}); err != nil {
		return result, err
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

	// Step 1 — reserve inventory
	progress.CurrentStep = "reserve-inventory"
	var itemID string
	if err := workflow.ExecuteActivity(ctx, a.ReserveInventory, input).Get(ctx, &itemID); err != nil {
		progress.Failed = "reserve-inventory"
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	compensations = append(compensations, func(c workflow.Context) error {
		return workflow.ExecuteActivity(c, a.ReleaseInventory, itemID).Get(c, nil)
	})
	progress.Completed = append(progress.Completed, "reserve-inventory")
	result.Confirmed = append(result.Confirmed, itemID)

	// Step 2 — charge payment
	progress.CurrentStep = "charge-payment"
	var paymentID string
	if err := workflow.ExecuteActivity(ctx, a.ChargePayment, input, itemID).Get(ctx, &paymentID); err != nil {
		progress.Failed = "charge-payment"
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	compensations = append(compensations, func(c workflow.Context) error {
		return workflow.ExecuteActivity(c, a.RefundPayment, paymentID, input.Amount).Get(c, nil)
	})
	progress.Completed = append(progress.Completed, "charge-payment")
	result.Confirmed = append(result.Confirmed, paymentID)

	// Step 3 — ship the order
	progress.CurrentStep = "ship-order"
	var trackingID string
	if err := workflow.ExecuteActivity(ctx, a.ShipOrder, input).Get(ctx, &trackingID); err != nil {
		progress.Failed = "ship-order"
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	compensations = append(compensations, func(c workflow.Context) error {
		return workflow.ExecuteActivity(c, a.CancelShipment, trackingID).Get(c, nil)
	})
	progress.Completed = append(progress.Completed, "ship-order")
	result.Confirmed = append(result.Confirmed, trackingID)

	// Step 4 — send confirmation
	progress.CurrentStep = "send-confirmation"
	var email string
	if err := workflow.ExecuteActivity(ctx, a.SendConfirmation, input).Get(ctx, &email); err != nil {
		progress.Failed = "send-confirmation"
		result.Status = "failed"
		runCompensations()
		return result, err
	}
	progress.Completed = append(progress.Completed, "send-confirmation")
	progress.CurrentStep = "done"
	result.Confirmed = append(result.Confirmed, email)
	result.Status = "completed"
	return result, nil
}
