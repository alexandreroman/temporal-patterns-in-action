<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Four-card summary derived from the event stream.
 *   - processed: unique indices whose final state is `completed`.
 *   - failed:    unique indices whose final state is `attempt_failed`
 *                (i.e. retried to exhaustion without completing).
 *   - throughput: processed / elapsed, where elapsed is first→last event time.
 */

interface Metrics {
  processed: number;
  failed: number;
  throughput: string;
}

const props = withDefaults(
  defineProps<{
    events: EventEnvelope[];
    total?: number;
  }>(),
  { total: 48 },
);

const metrics = computed<Metrics>(() => {
  const lastType = new Map<number, string>();

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const idx = data.index;
    if (typeof idx !== "number") continue;
    if (
      env.type === "batch.item.started" ||
      env.type === "batch.item.completed" ||
      env.type === "batch.item.attempt_failed"
    ) {
      lastType.set(idx, env.type);
    }
  }

  let processed = 0;
  let failed = 0;
  for (const type of lastType.values()) {
    if (type === "batch.item.completed") processed++;
    else if (type === "batch.item.attempt_failed") failed++;
  }

  const first = props.events[0];
  const last = props.events[props.events.length - 1];
  let elapsed = 0;
  if (first && last && first !== last) {
    const t0 = new Date(first.time).getTime();
    const t1 = new Date(last.time).getTime();
    if (!Number.isNaN(t0) && !Number.isNaN(t1) && t1 > t0) {
      elapsed = (t1 - t0) / 1000;
    }
  }

  const throughput = elapsed > 0 ? (processed / elapsed).toFixed(1) : "0.0";

  return { processed, failed, throughput };
});

interface Card {
  label: string;
  value: () => string | number;
}

const CARDS: readonly Card[] = [
  { label: "Total", value: () => props.total },
  { label: "Processed", value: () => metrics.value.processed },
  { label: "Failed", value: () => metrics.value.failed },
  { label: "Throughput (/s)", value: () => metrics.value.throughput },
];
</script>

<template>
  <div class="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
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
