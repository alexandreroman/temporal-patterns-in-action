package saga

import (
	"context"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "saga"

// Saga-specific business event types. Prefixed with the pattern name to
// guarantee no collision with types emitted by other patterns.
const (
	TypeInventoryReserved = "saga.inventory.reserved"
	TypeInventoryReleased = "saga.inventory.released"
	TypePaymentCharged    = "saga.payment.charged"
	TypePaymentRefunded   = "saga.payment.refunded"
	TypeOrderShipped      = "saga.shipping.shipped"
	TypeShipmentCancelled = "saga.shipping.cancelled"
	TypeConfirmationSent  = "saga.notification.sent"
	TypeEmailRetracted    = "saga.notification.retracted"
)

// publishBusiness emits a business event from within an activity. Nil-safe so
// tests that don't wire a publisher still pass.
func (a *Activities) publishBusiness(ctx context.Context, typ string, data any) {
	if a == nil || a.Publisher == nil {
		return
	}
	info := activity.GetInfo(ctx)
	env := events.NewEnvelope(Pattern,
		info.WorkflowExecution.ID, info.WorkflowExecution.RunID, typ, data)
	if err := a.Publisher.Publish(ctx, Pattern, env); err != nil {
		activity.GetLogger(ctx).Warn("publish business event failed",
			"type", typ, "error", err)
	}
}
