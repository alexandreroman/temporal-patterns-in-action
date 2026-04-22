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
  "check-fraud": "Check fraud",
  "release-fraud-hold": "Release fraud hold",
  "prepare-shipment": "Prepare shipment",
  "cancel-shipment": "Cancel shipment",
  "charge-customer": "Charge customer",
  "refund-customer": "Refund customer",
  "send-confirmation": "Send confirmation",
};

const COMP_STEPS = new Set(["release-fraud-hold", "cancel-shipment", "refund-customer"]);

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
      case "progress.step.started":
        tone = "running";
        if (COMP_STEPS.has(step)) compensating = true;
        message = compensating ? `Compensating: ${label}` : `Activity: ${label}`;
        break;
      case "progress.step.completed":
        message =
          compensating || COMP_STEPS.has(step) ? `${label} — reverted` : `${label} — completed`;
        break;
      case "progress.step.failed":
        // Don't flip `compensating` here: the worker interceptor emits
        // progress.step.failed on every retry attempt, so a retriable
        // timeout would be indistinguishable from a terminal failure.
        tone = "error";
        message = `${label} failed`;
        break;
      case "progress.workflow.completed":
        tone = "success";
        message = "Saga completed";
        break;
      case "progress.workflow.failed":
        tone = "error";
        message = compensating ? "Saga compensated" : "Workflow failed";
        break;
    }
  }

  return { tone, message };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
