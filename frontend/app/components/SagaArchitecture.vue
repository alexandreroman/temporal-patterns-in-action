<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { ArchState, EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Saga architecture: UI -> Temporal -> Worker -> (Inventory | Payment |
 *                                                  Shipping | Notification)
 * Maps saga step names to the four generic service slots `s1..s4` and folds
 * the event stream into the shared ArchState consumed by ArchitectureDiagram.
 */

const STEP_TO_SVC: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  "reserve-inventory": { node: "s1", edge: "wk_s1" },
  "release-inventory": { node: "s1", edge: "wk_s1" },
  "charge-payment": { node: "s2", edge: "wk_s2" },
  "refund-payment": { node: "s2", edge: "wk_s2" },
  "ship-order": { node: "s3", edge: "wk_s3" },
  "cancel-shipment": { node: "s3", edge: "wk_s3" },
  "send-confirmation": { node: "s4", edge: "wk_s4" },
};

const COMP_STEPS = new Set(["release-inventory", "refund-payment", "cancel-shipment"]);

const props = defineProps<{
  events: EventEnvelope[];
}>();

const arch = computed<ArchState>(() => {
  const nodes = initialNodes();
  const edges = initialEdges();

  let compensating = false;
  // Running stays true as long as we've seen at least one event and no
  // terminal event has closed the run. The worker no longer emits a
  // workflow.started signal — the Nuxt SSE endpoint synthesises only
  // the terminal events — so the first observed event anchors the run.
  let running = props.events.length > 0;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;

    switch (env.type) {
      case "progress.step.started": {
        const step = String(data.step ?? "");
        const svc = STEP_TO_SVC[step];
        if (COMP_STEPS.has(step)) compensating = true;
        if (svc) {
          resetServices(nodes, edges);
          nodes.temporal = "active";
          nodes.worker = compensating ? "warn" : "active";
          edges[svc.edge] = compensating ? "warn" : "active";
          nodes[svc.node] = compensating ? "warn" : "active";
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
        // Don't flip `compensating` here: the worker interceptor emits
        // progress.step.failed on every retry attempt, so a retriable
        // timeout would be indistinguishable from a terminal failure.
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

  // Keep the UI→Temporal→Worker strip lit while the run is in flight, since
  // no workflow.started event arrives to set it up explicitly.
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
    :service-labels="['Inventory', 'Payment', 'Shipping', 'Notification']"
    worker-label="Saga logic"
    label="Saga architecture diagram"
  />
</template>
