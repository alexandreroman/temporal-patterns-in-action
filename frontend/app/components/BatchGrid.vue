<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Grid of `total` cells, one per item. Each cell's state is folded from the
 * event stream: the most recent `batch.item.*` event for that index decides
 * the cell colour. A `queued` cell means the child workflow is already Running
 * in Temporal but is waiting for a worker activity slot, since throttling is
 * enforced via `worker.Options.MaxConcurrentActivityExecutionSize` rather than
 * by staggering workflow dispatch. Retry policy guarantees every item
 * eventually completes, so a terminal `failed` state is not surfaced.
 */

type CellState = "queued" | "running" | "retrying" | "done";

const props = withDefaults(
  defineProps<{
    events: EventEnvelope[];
    total?: number;
  }>(),
  { total: 48 },
);

const cells = computed<CellState[]>(() => {
  const out: CellState[] = Array.from({ length: props.total }, () => "queued");

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const idxRaw = data.index;
    if (typeof idxRaw !== "number" || idxRaw < 0 || idxRaw >= props.total) continue;

    switch (env.type) {
      case "batch.item.started": {
        const attempt = typeof data.attempt === "number" ? data.attempt : 1;
        out[idxRaw] = attempt > 1 ? "retrying" : "running";
        break;
      }
      case "batch.item.completed":
        out[idxRaw] = "done";
        break;
    }
  }

  return out;
});

const doneCount = computed(() => cells.value.filter((c) => c === "done").length);

const progressPct = computed(() =>
  props.total > 0 ? Math.round((doneCount.value / props.total) * 100) : 0,
);

const CELL_CLASS: Record<CellState, string> = {
  queued: "bg-slate-100 border-slate-200 dark:bg-slate-800 dark:border-slate-700",
  running: "bg-blue-100 border-blue-300 dark:bg-blue-900 dark:border-blue-600",
  retrying: "bg-amber-100 border-amber-300 dark:bg-amber-900 dark:border-amber-600",
  done: "bg-emerald-100 border-emerald-300 dark:bg-emerald-900 dark:border-emerald-600",
};

interface LegendEntry {
  state: CellState;
  label: string;
}

const LEGEND: readonly LegendEntry[] = [
  { state: "queued", label: "Queued" },
  { state: "running", label: "Running" },
  { state: "retrying", label: "Retry" },
  { state: "done", label: "Done" },
];
</script>

<template>
  <div>
    <div
      class="mb-1.5 flex items-center justify-between text-xs text-slate-500 dark:text-slate-400"
    >
      <span>{{ total }} images</span>
      <span class="tabular-nums">{{ progressPct }}%</span>
    </div>

    <div class="grid gap-[3px]" style="grid-template-columns: repeat(16, minmax(0, 1fr))">
      <div
        v-for="(state, idx) in cells"
        :key="idx"
        class="aspect-square rounded-[3px] border transition-colors duration-300"
        :class="CELL_CLASS[state]"
      />
    </div>

    <div class="mt-2 flex flex-wrap gap-3 text-[11px] text-slate-500 dark:text-slate-400">
      <div v-for="entry in LEGEND" :key="entry.state" class="flex items-center gap-1.5">
        <span class="size-2.5 rounded-[2px] border" :class="CELL_CLASS[entry.state]" />
        <span>{{ entry.label }}</span>
      </div>
    </div>
  </div>
</template>
