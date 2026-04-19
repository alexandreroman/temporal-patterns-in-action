import type { MaybeRefOrGetter } from "vue";
import { ref, toValue, watch, onBeforeUnmount } from "vue";
import { isEventEnvelope, type EventEnvelope } from "~~/shared/events";

export type StreamStatus = "idle" | "connecting" | "open" | "closed" | "error";

export function usePatternStream(
  pattern: MaybeRefOrGetter<string>,
  workflowId: MaybeRefOrGetter<string | null>,
) {
  const events = ref<EventEnvelope[]>([]);
  const status = ref<StreamStatus>("idle");

  let source: EventSource | null = null;
  const seen = new Set<string>();

  function close() {
    if (source !== null) {
      source.close();
      source = null;
    }
  }

  function open(url: string) {
    close();
    status.value = "connecting";
    source = new EventSource(url);

    source.onopen = () => {
      status.value = "open";
    };
    source.onerror = () => {
      status.value = source?.readyState === EventSource.CLOSED ? "closed" : "error";
    };
    source.onmessage = (event) => {
      ingest(event.data);
    };
  }

  function ingest(raw: string) {
    if (!raw) return;
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      return;
    }
    if (!isEventEnvelope(parsed)) return;
    if (seen.has(parsed.id)) return;
    seen.add(parsed.id);
    events.value = [...events.value, parsed];
  }

  watch(
    () => [toValue(pattern), toValue(workflowId)] as const,
    ([nextPattern, nextId]) => {
      events.value = [];
      seen.clear();
      if (!import.meta.client) return;
      if (!nextPattern || !nextId) {
        close();
        status.value = "idle";
        return;
      }
      open(`/api/patterns/${encodeURIComponent(nextPattern)}/${encodeURIComponent(nextId)}/events`);
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    close();
    status.value = "closed";
  });

  function waitForOpen(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (status.value === "open") return resolve();
      const stop = watch(status, (s) => {
        if (s === "open") {
          stop();
          resolve();
        } else if (s === "error" || s === "closed") {
          stop();
          reject(new Error(`event stream ${s}`));
        }
      });
    });
  }

  return { events, status, waitForOpen };
}
