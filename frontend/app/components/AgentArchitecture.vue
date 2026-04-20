<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { ArchState, Edges, EdgeKey, Nodes, NodeKey } from "~/types/architecture";

/**
 * Agent architecture: UI -> Temporal -> Worker -> (LLM | Flights | Hotels |
 *                                                  Calendar MCP). The active
 * service is derived from the tool name on `agent.tool.*` events, since
 * `execute-mcp-tool` is a single activity and carries no routing on its own.
 */

const NODE_IDS: NodeKey[] = ["ui", "temporal", "worker", "s1", "s2", "s3", "s4"];
const EDGE_IDS: EdgeKey[] = ["ui_tmp", "tmp_wk", "wk_s1", "wk_s2", "wk_s3", "wk_s4"];
const SERVICE_NODES: NodeKey[] = ["s1", "s2", "s3", "s4"];
const SERVICE_EDGES: EdgeKey[] = ["wk_s1", "wk_s2", "wk_s3", "wk_s4"];

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

function initialNodes(): Nodes {
  return {
    ui: "idle",
    temporal: "idle",
    worker: "idle",
    s1: "idle",
    s2: "idle",
    s3: "idle",
    s4: "idle",
  };
}
function initialEdges(): Edges {
  return {
    ui_tmp: "idle",
    tmp_wk: "idle",
    wk_s1: "idle",
    wk_s2: "idle",
    wk_s3: "idle",
    wk_s4: "idle",
  };
}

function resetAll(nodes: Nodes, edges: Edges) {
  for (const id of NODE_IDS) nodes[id] = "idle";
  for (const id of EDGE_IDS) edges[id] = "idle";
}
function resetServices(nodes: Nodes, edges: Edges) {
  for (const id of SERVICE_NODES) {
    if (nodes[id] !== "ok" && nodes[id] !== "error") nodes[id] = "idle";
  }
  for (const id of SERVICE_EDGES) {
    if (edges[id] !== "error") edges[id] = "idle";
  }
}

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
        resetServices(nodes, edges);
        running = false;
        nodes.ui = "error";
        nodes.temporal = "error";
        nodes.worker = "error";
        edges.ui_tmp = "error";
        edges.tmp_wk = "error";
        break;
    }
  }

  if (running) {
    if (nodes.ui === "idle") nodes.ui = "active";
    if (nodes.temporal === "idle") nodes.temporal = "active";
    if (nodes.worker === "idle") nodes.worker = "active";
    if (edges.ui_tmp === "idle") edges.ui_tmp = "active";
    if (edges.tmp_wk === "idle") edges.tmp_wk = "active";
  }

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
