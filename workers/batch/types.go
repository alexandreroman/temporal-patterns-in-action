package batch

// TaskQueue is the Temporal task queue used by the long-running batch pattern worker.
const TaskQueue = "patterns-batch"

// BatchInput is the input to the batch workflow.
type BatchInput struct {
	BatchID     string  `json:"batchId"`
	Total       int     `json:"total"`
	Parallelism int     `json:"parallelism"`
	// FailureRate is the probability of a first-attempt retryable failure per
	// item (0..1). Retries are bounded so a hit eventually succeeds.
	FailureRate float64 `json:"failureRate"`
}

// BatchResult is the output of the batch workflow.
type BatchResult struct {
	BatchID   string `json:"batchId"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
}

// ImageItem is a single unit of work dispatched by the workflow.
type ImageItem struct {
	BatchID string `json:"batchId"`
	Index   int    `json:"index"`
	Service string `json:"service"` // one of: resize, thumbnail, cdn, metadata
}

// Progress is returned by the getProgress query handler.
type Progress struct {
	Total     int `json:"total"`
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
	InFlight  int `json:"inFlight"`
}
