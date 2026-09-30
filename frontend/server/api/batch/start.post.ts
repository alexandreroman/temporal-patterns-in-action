import { BATCH_TOTAL } from "~~/shared/constants";
import type { BatchStartRequest, BatchStartResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<BatchStartResponse> => {
  const body = await readBody<BatchStartRequest>(event);

  const client = await getTemporalClient();
  const workflowId = `batch-${body.batchId}`;
  const failureRate = body.scenario === "failures" ? 0.18 : 0;

  await client.workflow.start(BATCH_WORKFLOW_TYPE, {
    taskQueue: BATCH_TASK_QUEUE,
    workflowId,
    args: [
      {
        batchId: body.batchId,
        total: BATCH_TOTAL,
        failureRate,
      },
    ],
  });

  return { workflowId };
});
