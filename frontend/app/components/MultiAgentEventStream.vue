<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";
import type { DotColor } from "~/types/event-stream";

defineProps<{
  events: EventEnvelope[];
}>();

function truncate(s: string, n = 48): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = typeof data.step === "string" ? data.step : "";
  const error = typeof data.error === "string" ? data.error : "";
  const attempt = typeof data.attempt === "number" ? data.attempt : null;
  const topicName = typeof data.topicName === "string" ? data.topicName : "";
  const query = typeof data.query === "string" ? data.query : "";
  const sourcesFound = typeof data.sourcesFound === "number" ? data.sourcesFound : null;

  switch (env.type) {
    case "progress.workflow.completed":
      return "Workflow completed";
    case "progress.workflow.failed":
      return `Workflow failed: ${error}`;
    case "progress.step.started":
      return `${step} started${attempt ? ` (attempt ${attempt})` : ""}`;
    case "progress.step.completed":
      return `${step} completed`;
    case "progress.step.failed":
      return `${step} failed: ${error}`;
    case "multi-agent.user.prompt":
      return `User prompt received`;
    case "multi-agent.plan.ready": {
      const subs = Array.isArray(data.subtopics) ? data.subtopics.length : 0;
      return `Plan ready — ${subs} subtopics`;
    }
    case "multi-agent.queries.ready": {
      const topics = Array.isArray(data.queries) ? data.queries.length : 0;
      return `Queries ready — ${topics} topics`;
    }
    case "multi-agent.fanout.started": {
      const agents = typeof data.agents === "number" ? data.agents : null;
      return `Fan-out: launching ${agents ?? "N"} child workflows`;
    }
    case "multi-agent.search.started":
      return `Search started — "${truncate(query)}"${attempt && attempt > 1 ? ` (attempt ${attempt})` : ""}`;
    case "multi-agent.search.completed":
      return `Search done — ${sourcesFound ?? "?"} sources`;
    case "multi-agent.search.failed":
      return `Search failed — "${truncate(query)}": ${error}`;
    case "multi-agent.child.completed":
      return `Child "${topicName}" ${data.partial ? "partial" : "done"} — ${sourcesFound ?? 0} sources`;
    case "multi-agent.child.failed":
      return `Child "${topicName}" failed: ${error}`;
    case "multi-agent.report.ready": {
      const used = typeof data.sourcesUsed === "number" ? data.sourcesUsed : 0;
      return `Report ready — ${used} citations`;
    }
    default:
      return env.type;
  }
}

function dotColor(env: EventEnvelope): DotColor {
  const t = env.type;
  if (t === "multi-agent.child.completed") {
    const data = env.data as Record<string, unknown>;
    return data.partial ? "amber" : "green";
  }
  if (t.endsWith(".failed") || t.endsWith("workflow.failed")) return "red";
  if (t === "progress.step.failed") return "amber";
  if (t.endsWith(".completed") || t.endsWith(".ready")) return "green";
  if (t.endsWith(".started") || t === "multi-agent.user.prompt") return "blue";
  return "blue";
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
