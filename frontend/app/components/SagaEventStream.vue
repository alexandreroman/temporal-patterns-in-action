<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const reversed = computed(() => [...props.events].reverse());

type DotColor = "blue" | "green" | "red" | "amber";

function dotColor(type: string): DotColor {
  if (type.includes("started")) return "blue";
  if (type.includes("completed")) return "green";
  if (type.includes("failed")) return "red";
  if (type.includes("compensation") || type.includes("cancelled") || type.includes("refunded"))
    return "amber";
  return "blue";
}

const DOT_CLS: Record<DotColor, string> = {
  blue: "bg-blue-500",
  green: "bg-emerald-500",
  red: "bg-rose-500",
  amber: "bg-amber-500",
};

function shortType(type: string): string {
  return type.replace(/^(progress|saga)\./, "");
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? iso
    : d.toLocaleTimeString([], { hour12: false, fractionalSecondDigits: 1 });
}

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = data.step ? String(data.step) : "";
  const error = data.error ? String(data.error) : "";

  switch (env.type) {
    case "progress.workflow.started":
      return "Workflow started";
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
    case "progress.compensation.started":
      return "Compensations started";
    case "progress.compensation.completed":
      return "Compensations completed";
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
    case "saga.notification.retracted":
      return `Email retracted from ${data.email}`;
    default:
      if (env.type.startsWith("saga.")) {
        return shortType(env.type);
      }
      return env.type;
  }
}
</script>

<template>
  <div
    class="flex flex-col overflow-hidden rounded-xl border border-slate-200 dark:border-slate-700"
  >
    <div
      class="border-b border-slate-200 px-4 py-2 text-xs font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300"
    >
      Event stream
    </div>
    <div class="max-h-72 flex-1 overflow-y-auto px-3 py-2">
      <p v-if="events.length === 0" class="py-4 text-center text-xs text-slate-400">
        No events yet.
      </p>
      <div
        v-for="env in reversed"
        :key="env.id"
        class="flex items-baseline gap-2 border-b border-slate-100 py-1.5 text-[11px] last:border-0 dark:border-slate-800"
      >
        <span class="mt-1 size-1.5 shrink-0 rounded-full" :class="DOT_CLS[dotColor(env.type)]" />
        <span class="shrink-0 font-mono text-[10px] text-slate-400">
          {{ formatTime(env.time) }}
        </span>
        <span class="text-slate-600 dark:text-slate-400">
          {{ eventLabel(env) }}
        </span>
      </div>
    </div>
  </div>
</template>
