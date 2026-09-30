import { computed, ref } from "vue";
import { isTerminalEvent } from "~~/shared/events";

/**
 * Run lifecycle shared by every pattern page: owns the workflow id, the
 * pattern's event stream, and the starting/running/error state around a
 * start request.
 */
export function usePatternRun(pattern: string) {
  const workflowId = ref<string | null>(null);
  const starting = ref(false);
  const error = ref<string | null>(null);

  const { events, waitForOpen } = usePatternStream(pattern, workflowId);

  const running = computed(() => {
    if (starting.value) return true;
    if (!workflowId.value) return false;
    return !events.value.some(isTerminalEvent);
  });

  /**
   * Opens the event stream for `id`, then calls `startFn` to start the
   * workflow. Subscribing first matters: core NATS has no replay, and the
   * first events fire almost immediately after the start request.
   */
  async function run(id: string, startFn: () => Promise<unknown>): Promise<void> {
    error.value = null;
    starting.value = true;
    // waitForOpen() must follow this assignment in the same tick: the stream
    // watch (flush: "sync") resets the connection status inside the setter,
    // so waitForOpen() waits for the new stream instead of the previous one.
    workflowId.value = id;
    try {
      await waitForOpen();
      await startFn();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      workflowId.value = null;
    } finally {
      starting.value = false;
    }
  }

  return { workflowId, events, starting, running, error, run };
}
