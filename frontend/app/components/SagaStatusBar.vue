<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

type Tone = "idle" | "running" | "success" | "error";

interface Derived {
  tone: Tone;
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

  let tone: Tone = "idle";
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

const dotClass: Record<Tone, string> = {
  idle: "bg-slate-400",
  running: "bg-blue-500 animate-pulse",
  success: "bg-emerald-500",
  error: "bg-rose-500",
};
</script>

<template>
  <div
    class="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-300"
  >
    <span class="size-2 shrink-0 rounded-full" :class="dotClass[derived.tone]" />
    <span>{{ derived.message }}</span>
  </div>
</template>
