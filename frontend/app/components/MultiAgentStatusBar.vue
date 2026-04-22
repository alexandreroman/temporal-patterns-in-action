<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { StatusTone } from "~/types/status-bar";

interface Derived {
  tone: StatusTone;
  message: string;
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const derived = computed<Derived>(() => {
  if (props.events.length === 0) {
    return { tone: "idle", message: "Ready — pick a scenario and run" };
  }

  let planReady = false;
  let queriesReady = false;
  let fanoutStarted = false;
  let reportReady = false;
  let anySearchFailed = false;
  let completed = false;
  let failed = false;
  let failError = "";

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    switch (env.type) {
      case "multi-agent.plan.ready":
        planReady = true;
        break;
      case "multi-agent.queries.ready":
        queriesReady = true;
        break;
      case "multi-agent.fanout.started":
        fanoutStarted = true;
        break;
      case "multi-agent.search.failed":
        anySearchFailed = true;
        break;
      case "multi-agent.child.completed":
        if (data.partial) anySearchFailed = true;
        break;
      case "multi-agent.report.ready":
        reportReady = true;
        break;
      case "progress.workflow.completed":
        completed = true;
        break;
      case "progress.workflow.failed":
        failed = true;
        if (typeof data.error === "string") failError = data.error;
        break;
    }
  }

  if (failed) {
    return { tone: "error", message: `Research failed${failError ? `: ${failError}` : ""}` };
  }
  if (completed) {
    return {
      tone: "success",
      message: anySearchFailed ? "Research completed with partial results" : "Research completed",
    };
  }
  if (reportReady) {
    return { tone: "running", message: "Finalising report…" };
  }
  if (fanoutStarted) {
    return {
      tone: "running",
      message: anySearchFailed
        ? "Running 3 agents — tolerating partial failures…"
        : "Running 3 research agents in parallel…",
    };
  }
  if (queriesReady) {
    return { tone: "running", message: "Queries ready — fanning out…" };
  }
  if (planReady) {
    return { tone: "running", message: "Plan ready — generating queries…" };
  }
  return { tone: "running", message: "Planning research…" };
});
</script>

<template>
  <StatusBar :tone="derived.tone" :message="derived.message" />
</template>
