package encryption

// Task queues — one per scenario. Each worker binds to its own queue with its
// own data converter, so the same workflow code runs against clear or
// encrypted payloads depending on which queue the caller targets.
const (
	TaskQueueClear     = "patterns-encryption-clear"
	TaskQueueEncrypted = "patterns-encryption-encrypted"
)

// SensitiveOrder is the workflow input. Its fields deliberately carry
// personal data (name, email, card hint) so the UI can illustrate what an
// unencrypted Temporal history would leak.
type SensitiveOrder struct {
	OrderID  string   `json:"orderId"`
	Customer Customer `json:"customer"`
	Items    []Item   `json:"items"`
	Total    float64  `json:"total"`
}

// Customer is the PII part of the order.
type Customer struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	CardLast4 string `json:"cardLast4"`
}

// Item is a single line on the order.
type Item struct {
	SKU   string  `json:"sku"`
	Qty   int     `json:"qty"`
	Price float64 `json:"price"`
}

// OrderConfirmation is the workflow output.
type OrderConfirmation struct {
	OrderID     string `json:"orderId"`
	Status      string `json:"status"`
	PaymentRef  string `json:"paymentRef"`
	TrackingID  string `json:"trackingId"`
	ReceiptSent bool   `json:"receiptSent"`
}
