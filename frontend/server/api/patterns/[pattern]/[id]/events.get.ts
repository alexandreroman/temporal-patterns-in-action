import { randomUUID } from "node:crypto";
import type { WorkflowHandle } from "@temporalio/client";
import { subscribe } from "~~/server/utils/nats";
// Re-exported by the util rather than imported from "@temporalio/client":
// see the CommonJS loading note in server/utils/temporal.ts.
import { WorkflowNotFoundError } from "~~/server/utils/temporal";
import type { EventEnvelope } from "~~/shared/events";

const HEARTBEAT_INTERVAL_MS = 15_000;
const DESCRIBE_POLL_INTERVAL_MS = 250;
const DESCRIBE_POLL_DEADLINE_MS = 30_000;
const TERMINAL_POLL_INTERVAL_MS = 500;
// Both values end up in the NATS subject, so they must stay a single token.
const SUBJECT_TOKEN = /^[\w-]+$/;

export default defineEventHandler(async (event) => {
  const pattern = getRouterParam(event, "pattern");
  const id = getRouterParam(event, "id");

  if (!pattern || !id || !SUBJECT_TOKEN.test(pattern) || !SUBJECT_TOKEN.test(id)) {
    throw createError({ statusCode: 400, statusMessage: "invalid pattern or workflow id" });
  }

  const stream = createEventStream(event);

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

  // Workers never emit terminal workflow events: the SSE endpoint watches the
  // handle and synthesises them.
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

    // Poll describe() rather than awaiting handle.result(): the result payload
    // may be encrypted (e.g. the encryption pattern), and the plain Temporal
    // client has no PayloadCodec. describe() reports status without decoding
    // the payload, keeping this endpoint pattern-agnostic.
    const terminal =
      status.name === "RUNNING" ? await waitForTerminal(handle, isClosed) : status.name;
    if (!terminal || isClosed()) return;

    if (terminal === "COMPLETED") {
      await pushSynthetic(push, pattern, workflowId, runId, "progress.workflow.completed", {});
    } else {
      // FAILED / CANCELLED / TERMINATED / TIMED_OUT — surface as failure with
      // the status name so the UI reflects the outcome.
      await pushSynthetic(push, pattern, workflowId, runId, "progress.workflow.failed", {
        error: `workflow ${terminal.toLowerCase()}`,
      });
    }
  } catch (err) {
    console.error("sse: terminal-state watcher failed", { pattern, workflowId, err });
  }
}

async function waitForTerminal(
  handle: WorkflowHandle,
  isClosed: () => boolean,
): Promise<string | null> {
  while (!isClosed()) {
    try {
      const { status } = await handle.describe();
      if (status.name !== "RUNNING") return status.name;
    } catch {
      // Transient error — keep polling until the stream closes.
    }
    await sleep(TERMINAL_POLL_INTERVAL_MS);
  }
  return null;
}

async function waitForDescription(handle: WorkflowHandle, isClosed: () => boolean) {
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

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
