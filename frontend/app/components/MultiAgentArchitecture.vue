<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Multi-agent architecture: UI -> Temporal -> Worker (parent wf) ->
 * (LLM API | Web search | Doc retrieval | News API). The parent-side
 * LLM activities (plan, queries, synthesise) light up s1. The three
 * child workflows do two WebSearch activities each; with no semantic
 * distinction between them in the event contract, we spread per-query
 * traffic across s2/s3/s4 by `queryIndex % 3`.
 */

const SEARCH_SERVICES: Array<{ node: NodeKey; edge: EdgeKey }> = [
  { node: "s2", edge: "wk_s2" },
  { node: "s3", edge: "wk_s3" },
  { node: "s4", edge: "wk_s4" },
];

// Parent-side activities that call the LLM API (s1).
const LLM_STEPS = new Set(["plan-research", "generate-queries", "synthesize-report"]);

const props = defineProps<{
  events: EventEnvelope[];
}>();

const arch = computed(() =>
  foldArch(props.events, (env, nodes, edges) => {
    const data = env.data as Record<string, unknown>;
    const step = typeof data.step === "string" ? data.step : "";
    const qi = typeof data.queryIndex === "number" ? data.queryIndex : -1;
    const svc = qi >= 0 ? SEARCH_SERVICES[qi % SEARCH_SERVICES.length] : undefined;

    switch (env.type) {
      case "progress.step.started":
        if (LLM_STEPS.has(step)) {
          nodes.temporal = "active";
          nodes.worker = "active";
          nodes.s1 = "active";
          edges.wk_s1 = "active";
        }
        break;

      case "progress.step.completed":
        if (LLM_STEPS.has(step)) {
          nodes.s1 = "ok";
          edges.wk_s1 = "idle";
        }
        break;

      case "progress.step.failed":
        // Interceptor fires on every retry attempt — mark transient.
        if (LLM_STEPS.has(step)) {
          nodes.s1 = "warn";
          edges.wk_s1 = "warn";
        }
        break;

      case "multi-agent.search.started":
        if (svc) {
          nodes.temporal = "active";
          nodes.worker = "active";
          nodes[svc.node] = "active";
          edges[svc.edge] = "active";
        }
        break;

      case "multi-agent.search.completed":
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
        break;

      case "multi-agent.search.failed":
        // Partial failure is recoverable at the parent — amber, not error.
        if (svc) {
          nodes[svc.node] = "warn";
          edges[svc.edge] = "warn";
        }
        break;

      case "multi-agent.report.ready":
        nodes.s1 = "ok";
        edges.wk_s1 = "idle";
        break;
    }
  }),
);
</script>

<template>
  <ArchitectureDiagram
    :arch="arch"
    :service-labels="['LLM API', 'Web search', 'Doc retrieval', 'News API']"
    worker-label="Supervisor"
    label="Multi-agent deep research architecture diagram"
  />
</template>
