import type { AgentStartRequest, AgentStartResponse } from "~~/shared/types";

const DEFAULT_PROMPT =
  "Plan a 5-day trip to Tokyo in October. Budget: $3000. I like food, temples, and nightlife.";

export default defineEventHandler(async (event): Promise<AgentStartResponse> => {
  const body = await readBody<AgentStartRequest>(event);

  const client = await getTemporalClient();
  const workflowId = `agent-${body.runId}`;

  await client.workflow.start(AGENT_WORKFLOW_TYPE, {
    taskQueue: AGENT_TASK_QUEUE,
    workflowId,
    args: [
      {
        prompt: DEFAULT_PROMPT,
        scenario: body.scenario,
      },
    ],
  });

  return { workflowId };
});
