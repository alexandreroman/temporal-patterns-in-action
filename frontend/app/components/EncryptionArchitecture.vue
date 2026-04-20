<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { ArchState, Edges, EdgeKey, Nodes, NodeKey } from "~/types/architecture";

const NODE_IDS: NodeKey[] = ["ui", "temporal", "worker", "s1", "s2", "s3", "s4"];
const EDGE_IDS: EdgeKey[] = ["ui_tmp", "tmp_wk", "wk_s1", "wk_s2", "wk_s3", "wk_s4"];
const SERVICE_NODES: NodeKey[] = ["s1", "s2", "s3", "s4"];
const SERVICE_EDGES: EdgeKey[] = ["wk_s1", "wk_s2", "wk_s3", "wk_s4"];

const STEP_TO_SVC: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  "validate-order": { node: "s1", edge: "wk_s1" },
  "charge-card": { node: "s2", edge: "wk_s2" },
  "ship-order": { node: "s3", edge: "wk_s3" },
  "send-receipt": { node: "s4", edge: "wk_s4" },
};

const props = defineProps<{
  events: EventEnvelope[];
  scenario: "clear" | "encrypted";
}>();

const isEncrypted = computed(() => props.scenario === "encrypted");

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

  // No workflow.started event is emitted, so the first observed event anchors
  // the run; closes only when a terminal event arrives.
  let running = props.events.length > 0;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;

    switch (env.type) {
      case "progress.step.started": {
        const svc = STEP_TO_SVC[String(data.step ?? "")];
        if (svc) {
          resetServices(nodes, edges);
          nodes.temporal = "active";
          nodes.worker = "active";
          edges[svc.edge] = "active";
          nodes[svc.node] = "active";
        }
        break;
      }

      case "progress.step.completed": {
        const svc = STEP_TO_SVC[String(data.step ?? "")];
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
        break;
      }

      case "progress.step.failed": {
        // Fires on every retry attempt, so a retriable timeout is
        // indistinguishable from a terminal failure — don't flip anything fatal.
        const svc = STEP_TO_SVC[String(data.step ?? "")];
        if (svc) {
          nodes[svc.node] = "error";
          edges[svc.edge] = "error";
          nodes.worker = "error";
        }
        break;
      }

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

  // Light the UI→Temporal→Worker strip while in flight; no workflow.started
  // event arrives to do it explicitly.
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
    :service-labels="['Validator', 'Payment', 'Shipping', 'Email']"
    :worker-label="isEncrypted ? 'Codec-wrapped' : 'No codec'"
    :codec="isEncrypted ? 'AES-256-GCM' : undefined"
    label="Encryption architecture diagram"
  />
</template>
