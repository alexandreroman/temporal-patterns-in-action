<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";
import type { DotColor } from "~/types/event-stream";
import {
  priorityLevel,
  tenantById,
  type PriorityKey,
  type TenantId,
} from "~/utils/priority-fairness";

defineProps<{
  events: EventEnvelope[];
}>();

function tenantName(id: unknown): string {
  if (id === "mission-critical" || id === "enterprise" || id === "business") {
    return tenantById(id as TenantId).name;
  }
  return String(id ?? "?");
}

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const tenant = tenantName(data.tenant);
  const ticketId = data.ticketId ? String(data.ticketId) : "";
  const agent = data.agent ? String(data.agent) : "";
  const prio = priorityLevel(data.priority as PriorityKey).label;

  switch (env.type) {
    case "progress.workflow.completed":
      return "All queues drained";
    case "progress.workflow.failed":
      return "Run failed";
    case "helpdesk.ticket.assigned":
      return `${tenant} assigned ${ticketId} (${prio}) → ${agent}`;
    case "helpdesk.ticket.resolved":
      return `${tenant} resolved ${ticketId} (${prio}) by ${agent}`;
    case "helpdesk.incident.injected":
      return `P0 incident injected for ${tenant} (${ticketId})`;
    case "helpdesk.run.seeded": {
      const tenants = data.tenants as Record<string, unknown[]> | undefined;
      const total =
        tenants !== undefined
          ? Object.values(tenants).reduce(
              (sum, list) => sum + (Array.isArray(list) ? list.length : 0),
              0,
            )
          : 0;
      const fairness = data.fairnessOn === false ? "OFF" : "ON";
      return `Run seeded — ${total} tickets queued, fairness ${fairness}`;
    }
    default:
      return env.type;
  }
}

function dotColor(env: EventEnvelope): DotColor {
  switch (env.type) {
    case "helpdesk.ticket.resolved":
    case "progress.workflow.completed":
      return "green";
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
