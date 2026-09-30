<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { StatusState } from "~/types/status-bar";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const derived = computed<StatusState>(() => {
  if (props.events.length === 0) {
    return { tone: "idle", message: "Ready — pick a scenario and run" };
  }

  let fairnessOn = true;
  let resolved = 0;
  let completed = false;
  let failed = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    if (env.type === "helpdesk.run.seeded") {
      fairnessOn = data.fairnessOn !== false;
    } else if (env.type === "helpdesk.ticket.resolved") {
      resolved += 1;
    } else if (env.type === "progress.workflow.completed") {
      completed = true;
    } else if (env.type === "progress.workflow.failed") {
      failed = true;
    }
  }

  if (failed) return { tone: "error", message: "Run failed" };
  const onOff = fairnessOn ? "ON" : "OFF";
  if (completed) {
    return { tone: "success", message: `Drained — ${resolved} total resolutions` };
  }
  // Omit the live resolved count: StatusBar replays its entry animation on
  // every message change, so a count ticking per resolution would keep it flashing.
  return { tone: "running", message: `Running with fairness ${onOff}` };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
