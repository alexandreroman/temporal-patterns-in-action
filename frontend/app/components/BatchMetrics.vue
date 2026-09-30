<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Four-card summary derived from the event stream.
 *   - processed: unique indices whose latest state is `completed`.
 *   - failed:    unique indices whose latest state is an `attempt_failed` on
 *                the final attempt (retries exhausted without completing).
 *   - queued:    indices with no `batch.item.*` event yet — still waiting for
 *                a slot in the workflow's sliding window, as in `BatchGrid`.
 *                Items currently running are counted in none of the cards.
 */

// Matches MaximumAttempts in the stage retry policy (workers/batch/workflow.go):
// an attempt_failed below this attempt is only a backoff before the next retry.
const MAX_ATTEMPTS = 3;

interface Metrics {
  processed: number;
  failed: number;
  queued: number;
}

const props = defineProps<{
  events: EventEnvelope[];
  total: number;
}>();

interface LastItemEvent {
  type: string;
  attempt: number;
}

const metrics = computed<Metrics>(() => {
  const lastEvent = new Map<number, LastItemEvent>();

  for (const env of props.events) {
    if (!env.type.startsWith("batch.item.")) continue;
    const data = env.data as Record<string, unknown>;
    const idx = data.index;
    if (typeof idx !== "number") continue;
    const attempt = typeof data.attempt === "number" ? data.attempt : 1;
    lastEvent.set(idx, { type: env.type, attempt });
  }

  let processed = 0;
  let failed = 0;
  for (const event of lastEvent.values()) {
    if (event.type === "batch.item.completed") {
      processed++;
    } else if (event.type === "batch.item.attempt_failed" && event.attempt >= MAX_ATTEMPTS) {
      failed++;
    }
  }

  const admitted = lastEvent.size;
  const queued = Math.max(0, props.total - admitted);

  return { processed, failed, queued };
});

interface Card {
  label: string;
  value: () => string | number;
}

const CARDS: readonly Card[] = [
  { label: "Total", value: () => props.total },
  { label: "Queued", value: () => metrics.value.queued },
  { label: "Failed", value: () => metrics.value.failed },
  { label: "Processed", value: () => metrics.value.processed },
];
</script>

<template>
  <div class="grid grid-cols-4 gap-2.5">
    <div
      v-for="card in CARDS"
      :key="card.label"
      class="rounded-md border border-slate-200 bg-slate-50 px-3.5 py-2.5 dark:border-slate-700 dark:bg-slate-800/60"
    >
      <div class="text-[11px] text-slate-500 dark:text-slate-400">{{ card.label }}</div>
      <div class="text-xl font-medium tabular-nums text-slate-900 dark:text-slate-100">
        {{ card.value() }}
      </div>
    </div>
  </div>
</template>
