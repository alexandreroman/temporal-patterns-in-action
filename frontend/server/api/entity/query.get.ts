import type { EntityCartProgress, EntityQueryResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<EntityQueryResponse> => {
  const workflowId = getQuery(event).workflowId;
  if (typeof workflowId !== "string" || workflowId.length === 0) {
    throw createError({ statusCode: 400, statusMessage: "workflowId is required" });
  }

  const client = await getTemporalClient();
  const handle = client.workflow.getHandle(workflowId);

  const progress = await handle.query<EntityCartProgress>(ENTITY_QUERY_GET_CART);

  return { workflowId, progress };
});
