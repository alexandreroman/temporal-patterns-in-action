package saga

// TaskQueue is the Temporal task queue used by the saga pattern worker.
const TaskQueue = "patterns-saga"

// OrderInput is the input to the saga workflow.
type OrderInput struct {
	CustomerID    string `json:"customerId"`
	OrderID       string `json:"orderId"`
	Amount        int    `json:"amount"`
	TransactionID string `json:"transactionId"`
	// FailAt simulates a failure at a given step for demo purposes.
	// Accepted values: "", "inventory", "payment", "shipping", "notification".
	FailAt string `json:"failAt,omitempty"`
}

// OrderResult is the output of the saga workflow.
type OrderResult struct {
	OrderID   string   `json:"orderId"`
	Status    string   `json:"status"`
	Confirmed []string `json:"confirmed"`
}

// Progress is returned by the getProgress query handler.
type Progress struct {
	CurrentStep string   `json:"currentStep"`
	Completed   []string `json:"completed"`
	Failed      string   `json:"failed,omitempty"`
}
