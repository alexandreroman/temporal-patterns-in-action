export interface SagaStartRequest {
  customerId: string;
  orderId: string;
  amount: number;
  failAt?: "" | "fraud" | "shipment" | "charge" | "notification";
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

export interface MultiAgentStartRequest {
  runId: string;
  scenario: "happy" | "partial";
}

export interface MultiAgentStartResponse {
  workflowId: string;
}

export interface EncryptionStartResponse {
  workflowId: string;
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

export interface EntityCartItem {
  itemId: string;
  name: string;
  priceCents: number;
  qty: number;
}

export interface EntityStartRequest {
  cartId: string;
}

export interface EntityStartResponse {
  workflowId: string;
}

export type EntitySignalType = "addItem" | "updateQty" | "removeItem" | "checkout";

export interface EntityAddItemPayload {
  itemId: string;
  name: string;
  priceCents: number;
  qty: number;
}

export interface EntityUpdateQtyPayload {
  itemId: string;
  qty: number;
}

export interface EntityRemoveItemPayload {
  itemId: string;
}

export type EntitySignalPayload =
  | { type: "addItem"; payload: EntityAddItemPayload }
  | { type: "updateQty"; payload: EntityUpdateQtyPayload }
  | { type: "removeItem"; payload: EntityRemoveItemPayload }
  | { type: "checkout"; payload: Record<string, never> };

export interface EntitySignalRequest {
  workflowId: string;
  signal: EntitySignalPayload;
}

export interface EntitySignalResponse {
  workflowId: string;
  type: EntitySignalType;
}

export interface EntityCartProgress {
  cartId: string;
  items: EntityCartItem[];
  totalCents: number;
  signalsReceived: number;
  queriesAnswered: number;
  checkedOut: boolean;
  historyLength: number;
}

export interface EntityQueryResponse {
  workflowId: string;
  progress: EntityCartProgress;
}
