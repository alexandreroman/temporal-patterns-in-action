<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Entity architecture: UI -> Temporal -> Worker (Cart entity) -> (Catalog |
 *                                                                 Pricing |
 *                                                                 Checkout |
 *                                                                 Email)
 * Activities that touch a backing service light the matching slot. Signal-
 * only steps (update-qty, remove-item) still pulse the
 * worker/temporal pair without engaging any service.
 */

const STEP_TO_SVC: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  "validate-item": { node: "s1", edge: "wk_s1" },
  "price-item": { node: "s2", edge: "wk_s2" },
  "process-payment": { node: "s3", edge: "wk_s3" },
  "send-confirmation": { node: "s4", edge: "wk_s4" },
};

const props = defineProps<{
  events: EventEnvelope[];
}>();

const arch = computed(() =>
  foldArch(props.events, (env, nodes, edges) => {
    const data = env.data as Record<string, unknown>;
    const step = typeof data.step === "string" ? data.step : "";

    switch (env.type) {
      case "progress.step.started": {
        const svc = STEP_TO_SVC[step];
        if (svc) resetServices(nodes, edges);
        nodes.temporal = "active";
        nodes.worker = "active";
        if (svc) {
          nodes[svc.node] = "active";
          edges[svc.edge] = "active";
        }
        break;
      }

      case "progress.step.completed": {
        const svc = STEP_TO_SVC[step];
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
        break;
      }

      case "progress.step.failed": {
        const svc = STEP_TO_SVC[step];
        if (svc) {
          nodes[svc.node] = "error";
          edges[svc.edge] = "error";
          nodes.worker = "error";
        }
        break;
      }
    }
  }),
);
</script>

<template>
  <ArchitectureDiagram
    :arch="arch"
    :service-labels="['Catalog', 'Pricing', 'Checkout', 'Email']"
    worker-label="Cart entity"
    label="Entity workflow architecture diagram"
  />
</template>
