import type { AgentApprovalRequest, AgentApprovalResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<AgentApprovalResponse> => {
  const body = await readBody<AgentApprovalRequest>(event);

  const client = await getTemporalClient();
  const handle = client.workflow.getHandle(body.workflowId);

  await handle.signal(AGENT_APPROVAL_SIGNAL, { approved: body.approved });

  return { workflowId: body.workflowId, approved: body.approved };
});
