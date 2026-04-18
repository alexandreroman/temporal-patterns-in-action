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

const STEP_LABELS: Record<string, string> = {
  "reserve-inventory": "Reserve inventory",
  "release-inventory": "Release inventory",
  "charge-payment": "Charge payment",
  "refund-payment": "Refund payment",
  "ship-order": "Ship order",
  "cancel-shipment": "Cancel shipment",
  "send-confirmation": "Send confirmation",
  "retract-email": "Retract email",
};

const COMP_STEPS = new Set([
  "release-inventory",
  "refund-payment",
  "cancel-shipment",
  "retract-email",
]);

const derived = computed<Derived>(() => {
  if (props.events.length === 0) {
    return { tone: "idle", message: "Ready — choose a failure point and run" };
  }

  let tone: StatusTone = "idle";
  let message = "Ready";
  let compensating = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const step = String(data.step ?? "");
    const label = STEP_LABELS[step] ?? step;

    switch (env.type) {
      case "progress.workflow.started":
        tone = "running";
        message = "Starting workflow…";
        break;
      case "progress.step.started":
        tone = "running";
        message =
          compensating || COMP_STEPS.has(step) ? `Compensating: ${label}` : `Activity: ${label}`;
        break;
      case "progress.step.completed":
        message =
          compensating || COMP_STEPS.has(step) ? `${label} — reverted` : `${label} — completed`;
        break;
      case "progress.step.failed":
        tone = "error";
        message = `${label} failed`;
        break;
      case "progress.compensation.started":
        compensating = true;
        tone = "running";
        message = "Compensating…";
        break;
      case "progress.compensation.completed":
        compensating = false;
        tone = "error";
        message = "Saga compensated";
        break;
      case "progress.workflow.completed":
        tone = "success";
        message = "Saga completed";
        break;
      case "progress.workflow.failed":
        tone = "error";
        if (!message.startsWith("Saga")) message = "Workflow failed";
        break;
    }
  }

  return { tone, message };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
