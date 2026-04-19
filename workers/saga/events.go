package saga

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
)
