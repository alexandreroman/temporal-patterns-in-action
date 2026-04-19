package batch

// TaskQueue is the Temporal task queue used by the long-running batch pattern worker.
const TaskQueue = "patterns-batch"

// BatchInput is the input to the batch workflow.
type BatchInput struct {
	BatchID     string `json:"batchId"`
	Total       int    `json:"total"`
	Parallelism int    `json:"parallelism"`
	// FailureRate is the probability of a first-attempt retryable failure per
	// stage (0..1). Retries are bounded so a hit eventually succeeds.
	FailureRate float64 `json:"failureRate"`
}

// BatchResult is the output of the batch workflow.
type BatchResult struct {
	BatchID   string `json:"batchId"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
}

// ImageInput is the input to ProcessImageWorkflow — the per-image child
// workflow that fans out across the 4 pipeline stages.
type ImageInput struct {
	BatchID string `json:"batchId"`
	// RootWorkflowID is the parent workflow ID; stage activities route their
	// business events to the parent's NATS subject via this value so the UI
	// sees every item event on a single per-batch stream.
	RootWorkflowID string  `json:"rootWorkflowId"`
	Index          int     `json:"index"`
	FailureRate    float64 `json:"failureRate"`
}

// StageInput is the input to each pipeline stage activity.
type StageInput struct {
	BatchID        string  `json:"batchId"`
	RootWorkflowID string  `json:"rootWorkflowId"`
	Index          int     `json:"index"`
	Service        string  `json:"service"`
	FailureRate    float64 `json:"failureRate"`
}

// Progress is returned by the getProgress query handler.
type Progress struct {
	Total     int `json:"total"`
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
	InFlight  int `json:"inFlight"`
}
