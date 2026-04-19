export interface SagaStartRequest {
  customerId: string;
  orderId: string;
  amount: number;
  failAt?: "" | "inventory" | "payment" | "shipping" | "notification";
}

export interface SagaStartResponse {
  workflowId: string;
}

export interface BatchStartRequest {
  batchId: string;
  // "clean" → 0 failure rate; "failures" → small non-zero rate so retries are visible.
  scenario: "clean" | "failures";
}

export interface BatchStartResponse {
  workflowId: string;
}
