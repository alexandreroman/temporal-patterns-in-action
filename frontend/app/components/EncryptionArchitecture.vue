<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { EncryptionStartRequest } from "~~/shared/types";
import type { EdgeKey, NodeKey } from "~/types/architecture";

const STEP_TO_SVC: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  "validate-order": { node: "s1", edge: "wk_s1" },
  "charge-card": { node: "s2", edge: "wk_s2" },
  "ship-order": { node: "s3", edge: "wk_s3" },
  "send-receipt": { node: "s4", edge: "wk_s4" },
};

const props = defineProps<{
  events: EventEnvelope[];
  scenario: EncryptionStartRequest["scenario"];
}>();

const arch = computed(() =>
  foldArch(props.events, (env, nodes, edges) => {
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
    }
  }),
);
</script>

<template>
  <ArchitectureDiagram
    :arch="arch"
    :service-labels="['Validator', 'Payment', 'Shipping', 'Email']"
    :worker-label="scenario === 'encrypted' ? 'Codec-wrapped' : 'No codec'"
    :codec="scenario === 'encrypted' ? 'AES-256-GCM' : undefined"
    label="Encryption architecture diagram"
  />
</template>
