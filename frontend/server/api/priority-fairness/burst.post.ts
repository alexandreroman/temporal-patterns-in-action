import type {
  PriorityFairnessSignalRequest,
  PriorityFairnessSignalResponse,
} from "~~/shared/types";

export default defineEventHandler(async (event): Promise<PriorityFairnessSignalResponse> => {
  const body = await readBody<PriorityFairnessSignalRequest>(event);

  const client = await getTemporalClient();
  const handle = client.workflow.getHandle(body.workflowId);

  await handle.signal(PRIORITY_FAIRNESS_SIGNAL_BURST, {});

  return { workflowId: body.workflowId };
});
