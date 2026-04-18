package saga

import (
	"context"

	"go.temporal.io/sdk/activity"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "saga"

// Saga-specific domain event types.
const (
	DomainInventoryReserved = "domain.inventory.reserved"
	DomainInventoryReleased = "domain.inventory.released"
	DomainPaymentCharged    = "domain.payment.charged"
	DomainPaymentRefunded   = "domain.payment.refunded"
	DomainOrderShipped      = "domain.shipping.shipped"
	DomainShipmentCancelled = "domain.shipping.cancelled"
	DomainConfirmationSent  = "domain.notification.sent"
	DomainEmailRetracted    = "domain.notification.retracted"
)

// publishDomain emits a domain event from within an activity. Nil-safe so
// tests that don't wire a publisher still pass.
func (a *Activities) publishDomain(ctx context.Context, typ string, data any) {
	if a == nil || a.Publisher == nil {
		return
	}
	info := activity.GetInfo(ctx)
	env := events.NewEnvelope(Pattern,
		info.WorkflowExecution.ID, info.WorkflowExecution.RunID, typ, data)
	if err := a.Publisher.Publish(ctx, Pattern, env); err != nil {
		activity.GetLogger(ctx).Warn("publish domain event failed",
			"type", typ, "error", err)
	}
}
