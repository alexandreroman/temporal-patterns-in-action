<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";
import type { DotColor } from "~/types/event-stream";

defineProps<{
  events: EventEnvelope[];
}>();

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = data.step ? String(data.step) : "";
  const error = data.error ? String(data.error) : "";
  const attempt = typeof data.attempt === "number" ? data.attempt : null;

  switch (env.type) {
    case "progress.workflow.completed":
      return "Workflow completed";
    case "progress.workflow.failed":
      return `Workflow failed: ${error}`;
    case "progress.step.started":
      return `${step} started (attempt ${attempt ?? 1})`;
    case "progress.step.completed":
      return `${step} completed`;
    case "progress.step.failed":
      return `${step} failed: ${error}`;
    case "saga.fraud.checked":
      return `Fraud check cleared #${data.checkId}`;
    case "saga.fraud.released":
      return `Fraud hold released #${data.checkId}`;
    case "saga.shipment.prepared":
      return `Shipment prepared #${data.shipmentId}`;
    case "saga.shipment.cancelled":
      return `Shipment cancelled #${data.shipmentId}`;
    case "saga.customer.charged":
      return `Customer charged EUR${data.amount}`;
    case "saga.customer.refunded":
      return `Customer refunded EUR${data.amount}`;
    case "saga.notification.sent":
      return `Confirmation sent to ${data.email}`;
  }
  return env.type;
}

function dotColor(env: EventEnvelope): DotColor {
  const t = env.type;
  if (t.includes("started")) return "blue";
  if (t.includes("completed")) return "green";
  if (t.includes("failed")) return "red";
  if (
    t.includes("released") ||
    t.includes("cancelled") ||
    t.includes("refunded") ||
    t.includes("compensation")
  )
    return "amber";
  return "blue";
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
