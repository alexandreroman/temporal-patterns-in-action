<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Three-card summary derived from the event stream.
 *   - processed: unique indices whose final state is `completed`.
 *   - failed:    unique indices whose final state is `attempt_failed`
 *                (i.e. retried to exhaustion without completing).
 */

interface Metrics {
  processed: number;
  failed: number;
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

  return { processed, failed };
});

interface Card {
  label: string;
  value: () => string | number;
}

const CARDS: readonly Card[] = [
  { label: "Total", value: () => props.total },
  { label: "Processed", value: () => metrics.value.processed },
  { label: "Failed", value: () => metrics.value.failed },
];
</script>

<template>
  <div class="grid grid-cols-3 gap-2.5">
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
