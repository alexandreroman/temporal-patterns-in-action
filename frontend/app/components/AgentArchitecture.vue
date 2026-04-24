<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { ArchState, EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Agent architecture: UI -> Temporal -> Worker -> (LLM | Flights | Hotels |
 *                                                  Calendar MCP). The active
 * service is derived from the tool name on `agent.tool.*` events, since
 * `execute-mcp-tool` is a single activity and carries no routing on its own.
 */

const TOOL_TO_SVC: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  search_flights: { node: "s2", edge: "wk_s2" },
  book_flight: { node: "s2", edge: "wk_s2" },
  search_hotels: { node: "s3", edge: "wk_s3" },
  book_hotel: { node: "s3", edge: "wk_s3" },
  get_calendar: { node: "s4", edge: "wk_s4" },
  send_itinerary: { node: "s4", edge: "wk_s4" },
};

const props = defineProps<{
  events: EventEnvelope[];
}>();

const arch = computed<ArchState>(() => {
  const nodes = initialNodes();
  const edges = initialEdges();

  let running = props.events.length > 0;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    const step = typeof data.step === "string" ? data.step : "";
    const tool = typeof data.name === "string" ? data.name : "";

    switch (env.type) {
      case "progress.step.started":
        if (step === "call-llm") {
          resetServices(nodes, edges);
          nodes.temporal = "active";
          nodes.worker = "active";
          nodes.s1 = "active";
          edges.wk_s1 = "active";
        } else if (step === "execute-mcp-tool") {
          nodes.temporal = "active";
          nodes.worker = "active";
        }
        break;

      case "progress.step.completed":
        if (step === "call-llm") {
          nodes.s1 = "ok";
          edges.wk_s1 = "idle";
        }
        break;

      case "progress.step.failed":
        // Interceptor fires on every retry attempt, so treat as transient —
        // mark the LLM slot amber without flipping the workflow.
        if (step === "call-llm") {
          nodes.s1 = "warn";
          edges.wk_s1 = "warn";
        }
        break;

      case "agent.tool.started": {
        const svc = TOOL_TO_SVC[tool];
        if (svc) {
          resetServices(nodes, edges);
          nodes.temporal = "active";
          nodes.worker = "active";
          nodes[svc.node] = "active";
          edges[svc.edge] = "active";
        }
        break;
      }

      case "agent.tool.completed": {
        const svc = TOOL_TO_SVC[tool];
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
        break;
      }

      case "agent.approval.requested":
        nodes.worker = "warn";
        nodes.temporal = "warn";
        edges.tmp_wk = "warn";
        edges.ui_tmp = "warn";
        nodes.ui = "warn";
        break;

      case "agent.approval.received":
        nodes.worker = "active";
        nodes.temporal = "active";
        edges.tmp_wk = "active";
        edges.ui_tmp = "active";
        nodes.ui = "active";
        break;

      case "progress.workflow.completed":
        resetAll(nodes, edges);
        running = false;
        nodes.temporal = "ok";
        nodes.ui = "ok";
        break;

      case "progress.workflow.failed":
        applyWorkflowFailed(nodes, edges);
        running = false;
        break;
    }
  }

  if (running) applyRunningBaseline(nodes, edges);

  return { nodes, edges, running };
});
</script>

<template>
  <ArchitectureDiagram
    :arch="arch"
    :service-labels="['LLM API', 'Flights MCP', 'Hotels MCP', 'Calendar MCP']"
    worker-label="Agent loop"
    label="Durable AI agent architecture diagram"
  />
</template>
