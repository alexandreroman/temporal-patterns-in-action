package encryption

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "encryption"

// Business event types emitted by the encryption workflow's activities.
// Prefixed with the pattern name to guarantee no collision with other
// patterns sharing the same NATS bus.
const (
	TypeOrderValidated = "encryption.order.validated"
	TypeCardCharged    = "encryption.card.charged"
	TypeOrderShipped   = "encryption.order.shipped"
	TypeReceiptSent    = "encryption.receipt.sent"
	TypeOrderCompleted = "encryption.order.completed"
)
