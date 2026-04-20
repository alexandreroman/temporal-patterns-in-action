<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { StatusTone } from "~/types/status-bar";

interface Derived {
  tone: StatusTone;
  message: string;
}

const props = defineProps<{
  events: EventEnvelope[];
  total: number;
}>();

const derived = computed<Derived>(() => {
  if (props.events.length === 0) {
    return { tone: "idle", message: "Ready — pick a scenario and run" };
  }

  const doneByIndex = new Set<number>();
  let seenStart = false;
  let completed = false;
  let failed = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    if (env.type === "batch.item.started") seenStart = true;
    if (env.type === "batch.item.completed" && typeof data.index === "number") {
      doneByIndex.add(data.index);
    }
    if (env.type === "progress.workflow.completed") completed = true;
    if (env.type === "progress.workflow.failed") failed = true;
  }

  const done = doneByIndex.size;

  if (failed) return { tone: "error", message: "Batch failed" };
  if (completed) {
    return { tone: "success", message: `Batch completed — ${done}/${props.total}` };
  }
  if (seenStart) {
    return { tone: "running", message: `Processing ${done}/${props.total} items…` };
  }
  return { tone: "running", message: "Starting batch…" };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
