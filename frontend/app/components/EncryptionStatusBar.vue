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
  scenario: "clear" | "encrypted";
}>();

const derived = computed<Derived>(() => {
  if (props.events.length === 0) {
    return { tone: "idle", message: "Ready — pick a scenario and run" };
  }

  let seenStart = false;
  let completed = false;
  let failed = false;

  for (const env of props.events) {
    if (env.type === "progress.step.started") seenStart = true;
    if (env.type === "progress.workflow.completed") completed = true;
    if (env.type === "progress.workflow.failed") failed = true;
  }

  if (failed) return { tone: "error", message: "Workflow failed" };
  if (completed) {
    const message =
      props.scenario === "encrypted"
        ? "Order processed — payloads stayed encrypted end-to-end"
        : "Order processed — WARNING: payloads stored in cleartext";
    return { tone: "success", message };
  }
  if (seenStart) {
    return { tone: "running", message: "Processing sensitive order…" };
  }
  return { tone: "running", message: "Starting workflow…" };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
