package entity

// TaskQueue is the Temporal task queue used by the entity pattern worker.
const TaskQueue = "patterns-entity"

// Signal names used by clients to mutate the cart entity workflow.
const (
	SignalAddItem    = "addItem"
	SignalUpdateQty  = "updateQty"
	SignalRemoveItem = "removeItem"
	SignalCheckout   = "checkout"
)

// QueryGetCart is the query handler name returning the current cart Progress.
const QueryGetCart = "getCart"

// CartItem is one line in the shopping cart.
type CartItem struct {
	ItemID     string `json:"itemId"`
	Name       string `json:"name"`
	PriceCents int    `json:"priceCents"`
	Qty        int    `json:"qty"`
}

// CartState is the workflow input and the entity's persistent state.
type CartState struct {
	CartID          string     `json:"cartId"`
	Items           []CartItem `json:"items"`
	SignalsReceived int        `json:"signalsReceived"`
	QueriesAnswered int        `json:"queriesAnswered"`
	CheckedOut      bool       `json:"checkedOut"`
}

// AddItemSignal is the payload for the addItem signal.
type AddItemSignal struct {
	ItemID     string `json:"itemId"`
	Name       string `json:"name"`
	PriceCents int    `json:"priceCents"`
	Qty        int    `json:"qty"`
}

// UpdateQtySignal is the payload for the updateQty signal. A qty ≤ 0 removes
// the matching line.
type UpdateQtySignal struct {
	ItemID string `json:"itemId"`
	Qty    int    `json:"qty"`
}

// RemoveItemSignal is the payload for the removeItem signal.
type RemoveItemSignal struct {
	ItemID string `json:"itemId"`
}

// CheckoutSignal is empty in the demo; still a struct so payload typing is
// consistent with the other signals.
type CheckoutSignal struct{}

// Progress is what getCart returns.
type Progress struct {
	CartID          string     `json:"cartId"`
	Items           []CartItem `json:"items"`
	TotalCents      int        `json:"totalCents"`
	SignalsReceived int        `json:"signalsReceived"`
	QueriesAnswered int        `json:"queriesAnswered"`
	CheckedOut      bool       `json:"checkedOut"`
	HistoryLength   int        `json:"historyLength"`
}
