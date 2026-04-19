import { randomUUID } from "node:crypto";
import { WorkflowNotFoundError } from "@temporalio/client";
import { subscribe } from "~~/server/utils/nats";
import type { EventEnvelope } from "~~/shared/events";

const KNOWN_PATTERNS = new Set(["saga", "batch"]);
const HEARTBEAT_INTERVAL_MS = 15_000;
const DESCRIBE_POLL_INTERVAL_MS = 250;
const DESCRIBE_POLL_DEADLINE_MS = 30_000;

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

  let closed = false;
  stream.onClosed(() => {
    closed = true;
    clearInterval(heartbeat);
    unsubscribe?.();
  });

  // Synthesise terminal workflow events from Temporal: the worker no longer
  // emits progress.workflow.{completed,failed}, so the SSE endpoint watches
  // the handle and pushes a matching envelope when the workflow terminates.
  void watchTerminalState(pattern, id, () => closed, stream.push.bind(stream));

  return stream.send();
});

type PushFn = (message: { id?: string; data: string; event?: string }) => Promise<void>;

async function watchTerminalState(
  pattern: string,
  workflowId: string,
  isClosed: () => boolean,
  push: PushFn,
): Promise<void> {
  try {
    const client = await getTemporalClient();
    const handle = client.workflow.getHandle(workflowId);

    const description = await waitForDescription(handle, isClosed);
    if (!description) return;

    const { runId, status } = description;

    if (status.name === "RUNNING") {
      try {
        await handle.result();
        if (isClosed()) return;
        await pushSynthetic(push, pattern, workflowId, runId, "progress.workflow.completed", {});
      } catch (err) {
        if (isClosed()) return;
        await pushSynthetic(push, pattern, workflowId, runId, "progress.workflow.failed", {
          error: extractErrorMessage(err),
        });
      }
      return;
    }

    if (isClosed()) return;
    if (status.name === "COMPLETED") {
      await pushSynthetic(push, pattern, workflowId, runId, "progress.workflow.completed", {});
    } else {
      // FAILED / CANCELLED / TERMINATED / TIMED_OUT — surface as failure with
      // the status name so the UI reflects the outcome.
      await pushSynthetic(push, pattern, workflowId, runId, "progress.workflow.failed", {
        error: `workflow ${status.name.toLowerCase()}`,
      });
    }
  } catch (err) {
    console.error("sse: terminal-state watcher failed", { pattern, workflowId, err });
  }
}

async function waitForDescription(
  handle: { describe: () => Promise<{ runId: string; status: { name: string } }> },
  isClosed: () => boolean,
) {
  const deadline = Date.now() + DESCRIBE_POLL_DEADLINE_MS;
  while (!isClosed() && Date.now() < deadline) {
    try {
      return await handle.describe();
    } catch (err) {
      if (!(err instanceof WorkflowNotFoundError)) throw err;
      await sleep(DESCRIBE_POLL_INTERVAL_MS);
    }
  }
  return null;
}

async function pushSynthetic(
  push: PushFn,
  pattern: string,
  workflowId: string,
  runId: string,
  type: string,
  data: Record<string, unknown>,
): Promise<void> {
  const envelope: EventEnvelope = {
    specversion: "1.0",
    id: randomUUID(),
    source: `patterns.${pattern}`,
    type,
    workflowId,
    runId,
    time: new Date().toISOString(),
    data,
  };
  await push({ id: envelope.id, data: JSON.stringify(envelope) });
}

function extractErrorMessage(err: unknown): string {
  if (err && typeof err === "object") {
    const cause = (err as { cause?: unknown }).cause;
    if (cause && typeof cause === "object" && typeof (cause as Error).message === "string") {
      return (cause as Error).message;
    }
    if (typeof (err as Error).message === "string") return (err as Error).message;
  }
  return String(err);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
