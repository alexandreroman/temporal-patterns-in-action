import type { EntitySignalRequest, EntitySignalResponse } from "~~/shared/types";

export default defineEventHandler(async (event): Promise<EntitySignalResponse> => {
  const body = await readBody<EntitySignalRequest>(event);
  const { workflowId, signal } = body;

  const client = await getTemporalClient();
  const handle = client.workflow.getHandle(workflowId);

  switch (signal.type) {
    case "addItem":
      await handle.signal(ENTITY_SIGNAL_ADD_ITEM, signal.payload);
      break;
    case "updateQty":
      await handle.signal(ENTITY_SIGNAL_UPDATE_QTY, signal.payload);
      break;
    case "removeItem":
      await handle.signal(ENTITY_SIGNAL_REMOVE_ITEM, signal.payload);
      break;
    case "checkout":
      await handle.signal(ENTITY_SIGNAL_CHECKOUT, {});
      break;
    default:
      throw createError({ statusCode: 400, statusMessage: "unknown signal" });
  }

  return { workflowId, type: signal.type };
});
