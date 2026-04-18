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
        data: JSON.stringify(envelope),
      });
    });
  } catch (error) {
    console.error("sse: failed to subscribe to NATS", { pattern, id, error });
    throw createError({ statusCode: 503, statusMessage: "event bus unavailable" });
  }

  // Push an immediate heartbeat so the response headers are flushed right
  // away. Without it, Node/h3 holds headers until the first chunk, which
  // delays EventSource.onopen on the client until the 15s interval fires.
  const pushHeartbeat = () => void stream.push({ data: "", event: "heartbeat" });
  pushHeartbeat();
  const heartbeat = setInterval(pushHeartbeat, HEARTBEAT_INTERVAL_MS);

  stream.onClosed(() => {
    clearInterval(heartbeat);
    unsubscribe?.();
  });

  return stream.send();
});
