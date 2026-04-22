package saga

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "saga"

// Saga-specific business event types. Prefixed with the pattern name to
// guarantee no collision with types emitted by other patterns.
const (
	TypeFraudChecked      = "saga.fraud.checked"
	TypeFraudReleased     = "saga.fraud.released"
	TypeShipmentPrepared  = "saga.shipment.prepared"
	TypeShipmentCancelled = "saga.shipment.cancelled"
	TypeCustomerCharged   = "saga.customer.charged"
	TypeCustomerRefunded  = "saga.customer.refunded"
	TypeConfirmationSent  = "saga.notification.sent"
)
