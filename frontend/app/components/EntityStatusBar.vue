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
}>();

const derived = computed<Derived>(() => {
  if (props.events.length === 0) {
    return { tone: "idle", message: "Ready — press Run scenario" };
  }

  let tone: StatusTone = "running";
  let message = "Starting cart…";

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const step = String(data.step ?? "");

    switch (env.type) {
      case "progress.step.started":
        tone = "running";
        switch (step) {
          case "validate-item":
            message = `Validating ${String(data.name ?? "item")}`;
            break;
          case "price-item":
            message = `Pricing ${String(data.name ?? "item")}`;
            break;
          case "update-qty":
            message = "Updating qty";
            break;
          case "remove-item":
            message = "Removing item";
            break;
          case "process-payment":
            message = "Processing payment";
            break;
          case "send-confirmation":
            message = "Sending confirmation";
            break;
          case "record-checkpoint":
            message = "Recording checkpoint";
            break;
          default:
            message = `${step}…`;
        }
        break;
      case "progress.workflow.completed":
        tone = "success";
        message = "Cart checked out";
        break;
      case "progress.workflow.failed":
        tone = "error";
        message = "Workflow failed";
        break;
    }
  }

  return { tone, message };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
