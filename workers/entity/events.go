package entity

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "entity"

// Entity-specific business event types. Prefixed with the pattern name to
// guarantee no collision with types emitted by other patterns.
const (
	TypeItemAdded        = "entity.item.added"
	TypeItemRemoved      = "entity.item.removed"
	TypeQtyUpdated       = "entity.qty.updated"
	TypePaymentProcessed = "entity.payment.processed"
	TypeConfirmationSent = "entity.confirmation.sent"
)
