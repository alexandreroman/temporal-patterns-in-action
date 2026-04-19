<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";

defineProps<{
  events: EventEnvelope[];
}>();

type DotColor = "blue" | "green" | "red" | "amber";

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = data.step ? String(data.step) : "";
  const error = data.error ? String(data.error) : "";

  switch (env.type) {
    case "progress.workflow.completed":
      return "Workflow completed";
    case "progress.workflow.failed":
      return `Workflow failed: ${error}`;
    case "progress.step.started":
      return `${step} started (attempt ${data.attempt})`;
    case "progress.step.completed":
      return `${step} completed`;
    case "progress.step.failed":
      return `${step} failed: ${error}`;
    case "saga.inventory.reserved":
      return `Inventory reserved #${data.itemId}`;
    case "saga.inventory.released":
      return `Inventory released #${data.itemId}`;
    case "saga.payment.charged":
      return `Payment charged EUR${data.amount}`;
    case "saga.payment.refunded":
      return `Payment refunded EUR${data.amount}`;
    case "saga.shipping.shipped":
      return `Order shipped #${data.trackingId}`;
    case "saga.shipping.cancelled":
      return `Shipment cancelled #${data.trackingId}`;
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
  if (t.includes("compensation") || t.includes("cancelled") || t.includes("refunded"))
    return "amber";
  return "blue";
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
