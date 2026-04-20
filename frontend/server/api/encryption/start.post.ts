import type {
  EncryptionStartRequest,
  EncryptionStartResponse,
  SensitiveOrder,
} from "~~/shared/types";

// Sample PII — kept in one place so the UI always renders the same card
// number / email that Temporal will store (or encrypt) for the run.
function sampleOrder(orderId: string): SensitiveOrder {
  return {
    orderId,
    customer: { name: "Alice Martin", email: "alice@example.com", cardLast4: "4242" },
    items: [
      { sku: "LAPTOP-PRO", qty: 1, price: 1299.99 },
      { sku: "USB-HUB-7", qty: 2, price: 34.99 },
    ],
    total: 1369.97,
  };
}

export default defineEventHandler(async (event): Promise<EncryptionStartResponse> => {
  const body = await readBody<EncryptionStartRequest>(event);

  const order = sampleOrder(body.orderId);
  const workflowId = `encryption-${body.orderId}`;

  const taskQueue =
    body.scenario === "encrypted" ? ENCRYPTION_TASK_QUEUE_ENCRYPTED : ENCRYPTION_TASK_QUEUE_CLEAR;
  const startingClient =
    body.scenario === "encrypted" ? await getEncryptedTemporalClient() : await getTemporalClient();

  const handle = await startingClient.workflow.start(ENCRYPTION_WORKFLOW_TYPE, {
    taskQueue,
    workflowId,
    args: [order],
  });

  // Fetch the stored payload through the plain client so the codec is NOT
  // applied on the way back — the point is to show what Temporal actually
  // keeps on disk. describe() first so we have a runId to pin the read.
  const plain = await getTemporalClient();
  const description = await handle.describe();
  const history = await plain.workflowService.getWorkflowExecutionHistory({
    namespace: process.env.TEMPORAL_NAMESPACE ?? "default",
    execution: { workflowId, runId: description.runId },
  });
  const firstPayload =
    history.history?.events?.[0]?.workflowExecutionStartedEventAttributes?.input?.payloads?.[0];
  const encoding = firstPayload?.metadata?.encoding
    ? Buffer.from(firstPayload.metadata.encoding).toString("utf8")
    : "unknown";
  const dataBase64 = firstPayload?.data ? Buffer.from(firstPayload.data).toString("base64") : "";

  return {
    workflowId,
    scenario: body.scenario,
    clientPayload: order,
    storedPayload: { encoding, dataBase64 },
  };
});
