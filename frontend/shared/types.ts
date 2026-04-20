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

export interface SensitiveOrder {
  orderId: string;
  customer: { name: string; email: string; cardLast4: string };
  items: Array<{ sku: string; qty: number; price: number }>;
  total: number;
}

export interface EncryptionStartRequest {
  orderId: string;
  scenario: "clear" | "encrypted";
}

export interface AgentStartRequest {
  runId: string;
  scenario: "happy" | "retry" | "approval";
}

export interface AgentStartResponse {
  workflowId: string;
}

export interface AgentApprovalRequest {
  workflowId: string;
  approved: boolean;
}

export interface AgentApprovalResponse {
  workflowId: string;
  approved: boolean;
}

export interface EncryptionStartResponse {
  workflowId: string;
  scenario: "clear" | "encrypted";
  /** Plaintext client payload — just the SensitiveOrder object echoed back. */
  clientPayload: SensitiveOrder;
  /**
   * How Temporal sees the workflow start input. `encoding` reports what's in
   * the payload metadata; `dataBase64` is the raw bytes of payload.data.
   */
  storedPayload: {
    encoding: string;
    dataBase64: string;
  };
}
