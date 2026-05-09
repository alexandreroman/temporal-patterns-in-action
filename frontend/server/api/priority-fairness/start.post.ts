import type { PriorityFairnessStartRequest, PriorityFairnessStartResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<PriorityFairnessStartResponse> => {
  const body = await readBody<PriorityFairnessStartRequest>(event);

  const client = await getTemporalClient();

  await client.workflow.start(PRIORITY_FAIRNESS_WORKFLOW_TYPE, {
    taskQueue: PRIORITY_FAIRNESS_TASK_QUEUE,
    workflowId: body.workflowId,
    args: [{ fairnessOn: body.fairnessOn }],
  });

  return { workflowId: body.workflowId };
});
