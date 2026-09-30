import type { PriorityFairnessStartRequest, PriorityFairnessStartResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<PriorityFairnessStartResponse> => {
  const body = await readBody<PriorityFairnessStartRequest>(event);

  const client = await getTemporalClient();

  await client.workflow.start(PRIORITY_FAIRNESS_WORKFLOW_TYPE, {
    taskQueue: PRIORITY_FAIRNESS_TASK_QUEUE,
    workflowId: body.workflowId,
    args: [{ fairnessOn: body.fairnessOn }],
    // The WaitTicketDone local activities keep the workflow task open, and the
    // Go SDK only heartbeats it at 0.8 x the task timeout: with the default 10s,
    // an inject-p0 signal could wait up to ~8s before the workflow sees it.
    workflowTaskTimeout: "2s",
  });

  return { workflowId: body.workflowId };
});
