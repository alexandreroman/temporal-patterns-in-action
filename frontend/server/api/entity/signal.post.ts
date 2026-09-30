import type { EntitySignalRequest, EntitySignalResponse, EntitySignalType } from "~~/shared/types";

const SIGNALS = {
  addItem: ENTITY_SIGNAL_ADD_ITEM,
  updateQty: ENTITY_SIGNAL_UPDATE_QTY,
  removeItem: ENTITY_SIGNAL_REMOVE_ITEM,
  checkout: ENTITY_SIGNAL_CHECKOUT,
} satisfies Record<EntitySignalType, string>;

export default defineEventHandler(async (event): Promise<EntitySignalResponse> => {
  const body = await readBody<EntitySignalRequest>(event);
  const { workflowId, signal } = body;

  // The body is untrusted JSON: the type union only holds for well-formed requests.
  if (!Object.hasOwn(SIGNALS, signal.type)) {
    throw createError({ statusCode: 400, statusMessage: "unknown signal" });
  }

  const client = await getTemporalClient();
  const handle = client.workflow.getHandle(workflowId);
  await handle.signal(SIGNALS[signal.type], signal.payload);

  return { workflowId, type: signal.type };
});
