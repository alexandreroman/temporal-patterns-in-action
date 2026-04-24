<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";
import type { DotColor } from "~/types/event-stream";

defineProps<{
  events: EventEnvelope[];
}>();

const STEP_LABELS: Record<string, string> = {
  "validate-order": "Validate",
  "charge-card": "Charge card",
  "ship-order": "Ship order",
  "send-receipt": "Send receipt",
};

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = data.step ? String(data.step) : "";
  const error = data.error ? String(data.error) : "";
  const attempt = typeof data.attempt === "number" ? data.attempt : null;
  const label = STEP_LABELS[step] ?? step;

  switch (env.type) {
    case "progress.workflow.completed":
      return "Workflow completed";
    case "progress.workflow.failed":
      return `Workflow failed: ${error}`;
    case "progress.step.started":
      return `${label} started (attempt ${attempt ?? 1})`;
    case "progress.step.completed":
      return `${label} completed`;
    case "progress.step.failed":
      return `${label} failed: ${error}`;
    case "encryption.order.validated":
      return `Order validated — ${data.items ?? "?"} items`;
    case "encryption.card.charged":
      return `Card •••${data.last4 ?? "????"} charged EUR${data.amount ?? "?"}`;
    case "encryption.order.shipped":
      return `Order shipped — ${data.trackingId ?? "?"}`;
    case "encryption.receipt.sent":
      return `Receipt sent to ${data.email ?? "?"}`;
    case "encryption.order.completed":
      return "Order completed";
    default:
      return env.type;
  }
}

const BUSINESS_SUCCESS: ReadonlySet<string> = new Set([
  "encryption.order.validated",
  "encryption.card.charged",
  "encryption.order.shipped",
  "encryption.receipt.sent",
]);

function dotColor(env: EventEnvelope): DotColor {
  const t = env.type;
  if (t.includes("failed")) return "red";
  if (t.includes("completed") || BUSINESS_SUCCESS.has(t)) return "green";
  return "blue";
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
