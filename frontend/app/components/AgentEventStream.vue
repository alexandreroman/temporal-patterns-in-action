<script setup lang="ts">
import type { EventEnvelope } from "~~/shared/events";

type DotColor = "blue" | "green" | "red" | "amber";

defineProps<{
  events: EventEnvelope[];
}>();

function eventLabel(env: EventEnvelope): string {
  const data = env.data as Record<string, unknown>;
  const step = typeof data.step === "string" ? data.step : "";
  const name = typeof data.name === "string" ? data.name : "";
  const error = typeof data.error === "string" ? data.error : "";
  const loop = typeof data.loop === "number" ? data.loop : null;
  const approved = Boolean(data.approved);

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
    case "agent.user.prompt":
      return "User submitted travel request";
    case "agent.llm.responded":
      return loop !== null ? `LLM responded (loop ${loop})` : "LLM responded";
    case "agent.tool.started":
      return `MCP tool started: ${name}`;
    case "agent.tool.completed":
      return `MCP tool completed: ${name}`;
    case "agent.approval.requested":
      return "LLM requests human approval";
    case "agent.approval.received":
      return approved ? "Approval signal received" : "Rejection signal received";
    case "agent.plan.ready":
      return "Final plan delivered";
  }
  return env.type;
}

function dotColor(env: EventEnvelope): DotColor {
  const t = env.type;
  if (t === "agent.approval.requested") return "amber";
  if (t === "agent.approval.received") {
    const data = env.data as Record<string, unknown>;
    return data.approved ? "green" : "red";
  }
  if (t === "agent.user.prompt" || t === "agent.llm.responded") return "blue";
  if (t === "agent.tool.started") return "blue";
  if (t === "agent.tool.completed" || t === "agent.plan.ready") return "green";
  if (t.includes("failed")) return "red";
  if (t.includes("completed")) return "green";
  if (t.includes("started")) return "blue";
  return "blue";
}
</script>

<template>
  <EventStream :events="events" :label-for="eventLabel" :dot-color="dotColor" />
</template>
