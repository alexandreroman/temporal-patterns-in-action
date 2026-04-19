import type { SagaStartRequest, SagaStartResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<SagaStartResponse> => {
  const body = await readBody<SagaStartRequest>(event);

  const client = await getTemporalClient();
  const workflowId = `saga-${body.orderId}`;

  await client.workflow.start(SAGA_WORKFLOW_TYPE, {
    taskQueue: SAGA_TASK_QUEUE,
    workflowId,
    args: [
      {
        customerId: body.customerId,
        orderId: body.orderId,
        amount: body.amount,
        transactionId: `tx-${body.orderId}`,
        failAt: body.failAt ?? "",
      },
    ],
  });

  return { workflowId };
});
