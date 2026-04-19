import type { BatchStartRequest, BatchStartResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<BatchStartResponse> => {
  const body = await readBody<BatchStartRequest>(event);

  if (!body?.batchId) {
    throw createError({ statusCode: 400, statusMessage: "batchId is required" });
  }

  const client = await getTemporalClient();
  const workflowId = `batch-${body.batchId}`;

  const total = body.total && body.total > 0 ? body.total : 48;
  const parallelism = body.parallelism && body.parallelism > 0 ? body.parallelism : 4;
  const failureRate = body.scenario === "failures" ? 0.18 : 0;

  await client.workflow.start(BATCH_WORKFLOW_TYPE, {
    taskQueue: BATCH_TASK_QUEUE,
    workflowId,
    args: [
      {
        batchId: body.batchId,
        total,
        parallelism,
        failureRate,
      },
    ],
  });

  return { workflowId };
});
