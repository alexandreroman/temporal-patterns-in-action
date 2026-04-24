<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { ArchState, EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Saga architecture: UI -> Temporal -> Worker -> (Fraud | Shipment |
 *                                                 Payment | Notification)
 * Maps saga step names to the four generic service slots `s1..s4` and folds
 * the event stream into the shared ArchState consumed by ArchitectureDiagram.
 */

const STEP_TO_SVC: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  "check-fraud": { node: "s1", edge: "wk_s1" },
  "release-fraud-hold": { node: "s1", edge: "wk_s1" },
  "prepare-shipment": { node: "s2", edge: "wk_s2" },
  "cancel-shipment": { node: "s2", edge: "wk_s2" },
  "charge-customer": { node: "s3", edge: "wk_s3" },
  "refund-customer": { node: "s3", edge: "wk_s3" },
  "send-confirmation": { node: "s4", edge: "wk_s4" },
};

const COMP_STEPS = new Set(["release-fraud-hold", "cancel-shipment", "refund-customer"]);

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
    :service-labels="['Fraud', 'Shipment', 'Payment', 'Notification']"
    worker-label="Saga logic"
    label="Saga architecture diagram"
  />
</template>
