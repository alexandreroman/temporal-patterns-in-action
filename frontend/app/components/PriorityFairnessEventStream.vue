<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";
import type { DotColor } from "~/types/event-stream";
import { tenantById, type TenantId } from "~/utils/priority-fairness";

defineProps<{
  events: EventEnvelope[];
}>();

function tenantName(id: unknown): string {
  if (id === "acme" || id === "brick" || id === "solo") {
    return tenantById(id as TenantId).name;
  }
  return String(id ?? "?");
}

function priorityLabel(key: unknown): string {
  const n = typeof key === "number" ? key : Number(key);
  if (!Number.isFinite(n)) return "P?";
  return `P${Math.max(0, n - 1)}`;
}

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const tenant = tenantName(data.tenantId);
  const ticketId = data.ticketId ? String(data.ticketId) : "";
  const agent = data.agent ? String(data.agent) : "";
  const prio = priorityLabel(data.priorityKey);

  switch (env.type) {
    case "progress.workflow.started":
      return `Workflow started — fairness ${data.fairnessOn ? "ON" : "OFF"}`;
    case "progress.workflow.completed":
      return "All queues drained";
    case "progress.workflow.failed":
      return "Run failed";
    case "helpdesk.ticket.assigned":
      return `${tenant} assigned ${ticketId} (${prio}) → ${agent}`;
    case "helpdesk.ticket.resolved":
      return `${tenant} resolved ${ticketId} (${prio}) by ${agent}`;
    case "helpdesk.dump.executed": {
      const count = typeof data.count === "number" ? data.count : 0;
      return `${tenant} flood — ${count} tickets queued`;
    }
    case "helpdesk.incident.injected":
      return `P0 incident injected for ${tenant} (${ticketId})`;
    default:
      return env.type;
  }
}

function dotColor(env: EventEnvelope): DotColor {
  switch (env.type) {
    case "progress.workflow.started":
    case "helpdesk.ticket.assigned":
      return "blue";
    case "helpdesk.ticket.resolved":
    case "progress.workflow.completed":
      return "green";
    case "helpdesk.dump.executed":
      return "amber";
    case "helpdesk.incident.injected":
    case "progress.workflow.failed":
      return "red";
    default:
      return "blue";
  }
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
