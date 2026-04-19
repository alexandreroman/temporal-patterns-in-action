import type { BatchStartRequest, BatchStartResponse } from "~~/shared/types";

const TOTAL = 48;
const PARALLELISM = 4;

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
        total: TOTAL,
        parallelism: PARALLELISM,
        failureRate,
      },
    ],
  });

  return { workflowId };
});
