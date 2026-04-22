import type { MultiAgentStartRequest, MultiAgentStartResponse } from "~~/shared/types";

const DEFAULT_PROMPT =
  "AI's impact on the EU labor market — risks, opportunities, and policy implications.";

export default defineEventHandler(async (event): Promise<MultiAgentStartResponse> => {
  const body = await readBody<MultiAgentStartRequest>(event);

  const client = await getTemporalClient();
  const workflowId = `multi-agent-${body.runId}`;

  await client.workflow.start(MULTI_AGENT_WORKFLOW_TYPE, {
    taskQueue: MULTI_AGENT_TASK_QUEUE,
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
