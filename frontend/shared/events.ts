// Frozen contract shared between Go workers and the Nuxt frontend.
// Subject format: patterns.<pattern>.<workflowId>.<category>

export const PROGRESS_TYPES = [
  "progress.workflow.started",
  "progress.workflow.completed",
  "progress.workflow.failed",
  "progress.step.started",
  "progress.step.completed",
  "progress.step.failed",
  "progress.compensation.started",
  "progress.compensation.completed",
] as const;

export type ProgressEventType = (typeof PROGRESS_TYPES)[number];

export const SAGA_DOMAIN_TYPES = [
  "domain.inventory.reserved",
  "domain.inventory.released",
  "domain.payment.charged",
  "domain.payment.refunded",
  "domain.shipping.shipped",
  "domain.shipping.cancelled",
  "domain.notification.sent",
  "domain.notification.retracted",
] as const;

export type SagaDomainEventType = (typeof SAGA_DOMAIN_TYPES)[number];

export type EventCategory = "progress" | "domain";

export interface EventEnvelope<T = unknown> {
  specversion: "1.0";
  id: string;
  source: string;
  type: string;
  workflowId: string;
  runId: string;
  time: string;
  data: T;
}

export type ProgressWorkflowStartedData = Record<string, never>;
export interface ProgressWorkflowCompletedData {
  durationMs: number;
}
export interface ProgressWorkflowFailedData {
  error: string;
}
export interface ProgressStepStartedData {
  step: string;
  attempt: number;
}
export interface ProgressStepCompletedData {
  step: string;
  attempt: number;
  durationMs: number;
}
export interface ProgressStepFailedData {
  step: string;
  attempt: number;
  error: string;
}
export type ProgressCompensationStartedData = Record<string, never>;
export type ProgressCompensationCompletedData = Record<string, never>;

export interface SagaInventoryReservedData {
  itemId: string;
}
export interface SagaInventoryReleasedData {
  itemId: string;
}
export interface SagaPaymentChargedData {
  amount: number;
}
export interface SagaPaymentRefundedData {
  amount: number;
}
export interface SagaOrderShippedData {
  trackingId: string;
}
export interface SagaShipmentCancelledData {
  trackingId: string;
}
export interface SagaConfirmationSentData {
  email: string;
}
export interface SagaEmailRetractedData {
  email: string;
}

export function categoryOf(type: string): EventCategory {
  const head = type.split(".", 1)[0];
  return head === "domain" ? "domain" : "progress";
}

export function isEventEnvelope(value: unknown): value is EventEnvelope {
  if (value === null || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  return (
    v.specversion === "1.0" &&
    typeof v.id === "string" &&
    typeof v.source === "string" &&
    typeof v.type === "string" &&
    typeof v.workflowId === "string" &&
    typeof v.runId === "string" &&
    typeof v.time === "string" &&
    typeof v.data === "object" &&
    v.data !== null
  );
}
