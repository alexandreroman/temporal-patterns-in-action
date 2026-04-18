export interface SagaStartRequest {
  customerId: string;
  orderId: string;
  amount: number;
  failAt?: "" | "inventory" | "payment" | "shipping" | "notification";
}

export interface SagaStartResponse {
  workflowId: string;
}

export interface SagaProgress {
  workflowId: string;
  status: "running" | "completed" | "failed";
  currentStep: string;
  completed: string[];
  failed?: string;
}
