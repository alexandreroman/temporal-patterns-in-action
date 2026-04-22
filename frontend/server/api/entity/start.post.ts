import type { EntityStartRequest, EntityStartResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<EntityStartResponse> => {
  const body = await readBody<EntityStartRequest>(event);

  const client = await getTemporalClient();
  const workflowId = `entity-${body.cartId}`;

  await client.workflow.start(ENTITY_WORKFLOW_TYPE, {
    taskQueue: ENTITY_TASK_QUEUE,
    workflowId,
    args: [
      {
        cartId: body.cartId,
        items: [],
        signalsReceived: 0,
        queriesAnswered: 0,
        checkedOut: false,
      },
    ],
  });

  return { workflowId };
});
