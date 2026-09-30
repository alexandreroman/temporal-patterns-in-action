import { AGENT_PROMPT } from "~~/shared/constants";
import type { AgentStartRequest, AgentStartResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<AgentStartResponse> => {
  const body = await readBody<AgentStartRequest>(event);

  const client = await getTemporalClient();
  const workflowId = `agent-${body.runId}`;

  await client.workflow.start(AGENT_WORKFLOW_TYPE, {
    taskQueue: AGENT_TASK_QUEUE,
    workflowId,
    args: [
      {
        prompt: AGENT_PROMPT,
        scenario: body.scenario,
      },
    ],
  });

  return { workflowId };
});
