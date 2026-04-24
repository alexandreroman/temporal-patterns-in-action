<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import { BATCH_SERVICE_LABEL } from "~/utils/batch-services";

/**
 * Sliding-window view: up to `parallelism` chips showing the items currently
 * occupying a worker activity slot. An item is "active" between its latest
 * `batch.item.started` and the NEXT `batch.item.stage_completed` /
 * `attempt_failed` / `completed` (or a workflow terminal event). Because a
 * stage_completed event releases the slot before the next stage re-acquires
 * one, the chips truly reflect the 4 items currently in a worker activity
 * slot. A chip flips to `retry` when a subsequent `started` arrives with
 * attempt>1.
 */

type SlotState = "active" | "retry";

interface ActiveItem {
  index: number;
  service: string;
  state: SlotState;
}

const props = withDefaults(
  defineProps<{
    events: EventEnvelope[];
    parallelism?: number;
  }>(),
  { parallelism: 4 },
);

const activeItems = computed<ActiveItem[]>(() => {
  const byIndex = new Map<number, ActiveItem>();
  let terminated = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;

    if (env.type === "progress.workflow.completed" || env.type === "progress.workflow.failed") {
      terminated = true;
      continue;
    }

    const idxRaw = data.index;
    if (typeof idxRaw !== "number") continue;
    const service = typeof data.service === "string" ? data.service : "";

    switch (env.type) {
      case "batch.item.started": {
        const attempt = typeof data.attempt === "number" ? data.attempt : 1;
        byIndex.set(idxRaw, {
          index: idxRaw,
          service,
          state: attempt > 1 ? "retry" : "active",
        });
        break;
      }
      case "batch.item.stage_completed":
      case "batch.item.attempt_failed":
      case "batch.item.completed":
        byIndex.delete(idxRaw);
        break;
    }
  }

  if (terminated) return [];
  // Iteration order preserves insertion — most recently started first.
  return Array.from(byIndex.values()).slice(0, props.parallelism);
});

function serviceLabel(service: string): string {
  return BATCH_SERVICE_LABEL[service] ?? service;
}

function chipText(item: ActiveItem): string {
  if (item.state === "retry") return `#${item.index} retry`;
  return `#${item.index} ${serviceLabel(item.service)}`;
}

const CHIP_ACTIVE =
  "border-blue-300 bg-blue-50 text-blue-700 " +
  "dark:border-blue-600 dark:bg-blue-950 dark:text-blue-200";
const CHIP_RETRY =
  "border-amber-300 bg-amber-50 text-amber-700 " +
  "dark:border-amber-600 dark:bg-amber-950 dark:text-amber-200";
const CHIP_IDLE =
  "border-slate-200 bg-slate-50 text-slate-400 " +
  "dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-500";

function chipClass(item: ActiveItem | null): string {
  if (!item) return CHIP_IDLE;
  if (item.state === "retry") return CHIP_RETRY;
  return CHIP_ACTIVE;
}

const slots = computed<(ActiveItem | null)[]>(() => {
  const out: (ActiveItem | null)[] = Array.from({ length: props.parallelism }, () => null);
  activeItems.value.forEach((item, i) => {
    if (i < out.length) out[i] = item;
  });
  return out;
});
</script>

<template>
  <div class="flex items-center gap-2">
    <div
      v-for="(item, idx) in slots"
      :key="idx"
      class="flex-1 rounded-md border px-2.5 py-1 text-center font-mono text-[11px] transition-colors duration-300"
      :class="chipClass(item)"
    >
      {{ item ? chipText(item) : "idle" }}
    </div>
  </div>
</template>
