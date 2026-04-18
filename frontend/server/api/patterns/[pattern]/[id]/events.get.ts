import { subscribe } from "~~/server/utils/nats";

const KNOWN_PATTERNS = new Set(["saga"]);
const HEARTBEAT_INTERVAL_MS = 15_000;

export default defineEventHandler(async (event) => {
  const pattern = getRouterParam(event, "pattern");
  const id = getRouterParam(event, "id");

  if (!pattern || !KNOWN_PATTERNS.has(pattern)) {
    throw createError({ statusCode: 404, statusMessage: "unknown pattern" });
  }
  if (!id) {
    throw createError({ statusCode: 400, statusMessage: "workflow id is required" });
  }

  const stream = createEventStream(event);
  setResponseHeader(event, "Cache-Control", "no-cache, no-transform");
  setResponseHeader(event, "Connection", "keep-alive");

  let unsubscribe: (() => void) | null = null;
  try {
    unsubscribe = await subscribe(`patterns.${pattern}.${id}.>`, (envelope) => {
      void stream.push({
        id: envelope.id,
        event: envelope.type,
        data: JSON.stringify(envelope),
      });
    });
  } catch (error) {
    console.error("sse: failed to subscribe to NATS", { pattern, id, error });
    throw createError({ statusCode: 503, statusMessage: "event bus unavailable" });
  }

  const heartbeat = setInterval(() => {
    void stream.push({ data: "", event: "heartbeat" });
  }, HEARTBEAT_INTERVAL_MS);

  stream.onClosed(() => {
    clearInterval(heartbeat);
    unsubscribe?.();
  });

  return stream.send();
});
