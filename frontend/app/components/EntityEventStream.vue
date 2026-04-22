<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";

type DotColor = "blue" | "green" | "red" | "amber";

defineProps<{
  events: EventEnvelope[];
}>();

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = data.step ? String(data.step) : "";
  const error = data.error ? String(data.error) : "";

  switch (env.type) {
    case "progress.workflow.completed":
      return "Cart checked out";
    case "progress.workflow.failed":
      return `Workflow failed: ${error}`;
    case "progress.step.started":
      return `${step} started (attempt ${data.attempt})`;
    case "progress.step.completed":
      return `${step} completed`;
    case "progress.step.failed":
      return `${step} failed: ${error}`;
    case "entity.item.added": {
      const price = Number(data.priceCents ?? 0) / 100;
      return `Added ${String(data.name ?? "")} × ${String(data.qty ?? 1)} ($${price.toFixed(2)})`;
    }
    case "entity.qty.updated":
      return `Updated ${String(data.itemId ?? "")} → × ${String(data.qty ?? 0)}`;
    case "entity.item.removed":
      return `Removed ${String(data.itemId ?? "")}`;
    case "entity.payment.processed":
      return `Payment processed — order ${String(data.orderId ?? "")}`;
    case "entity.confirmation.sent":
      return `Confirmation sent to ${String(data.email ?? "")}`;
    case "entity.query.getCart":
      return "Client queried getCart";
  }
  return env.type;
}

function dotColor(env: EventEnvelope): DotColor {
  const t = env.type;
  if (t === "entity.query.getCart") return "amber";
  if (t.includes("failed")) return "red";
  if (t.includes("completed")) return "green";
  if (t.includes("started")) return "blue";
  return "blue";
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
