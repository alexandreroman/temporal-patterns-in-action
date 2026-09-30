<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Batch architecture: UI -> Temporal -> Worker -> (Resize | Thumbnail | CDN |
 *                                                      Metadata DB)
 * The active service is derived from the `service` field that every stage
 * activity sets on its `batch.item.*` events, so the diagram does not need to
 * map activity names to services.
 */

const SERVICE_TO_NODE: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  resize: { node: "s1", edge: "wk_s1" },
  thumbnail: { node: "s2", edge: "wk_s2" },
  cdn: { node: "s3", edge: "wk_s3" },
  metadata: { node: "s4", edge: "wk_s4" },
};

const props = defineProps<{
  events: EventEnvelope[];
}>();

const arch = computed(() =>
  foldArch(props.events, (env, nodes, edges) => {
    const data = env.data as Record<string, unknown>;
    const svcKey = typeof data.service === "string" ? data.service : "";
    const svc = SERVICE_TO_NODE[svcKey];

    switch (env.type) {
      case "batch.item.started":
        if (svc) {
          nodes.temporal = "active";
          nodes.worker = "active";
          nodes[svc.node] = "active";
          edges[svc.edge] = "active";
        }
        break;

      case "batch.item.completed":
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
        break;

      case "batch.item.attempt_failed":
        // Transient — the retry will flip this back to active/ok.
        if (svc) {
          nodes[svc.node] = "warn";
          edges[svc.edge] = "warn";
        }
        break;

      case "batch.summary.reported":
        // The summary activity writes to the Metadata DB (s4).
        nodes.s4 = "active";
        edges.wk_s4 = "active";
        break;
    }
  }),
);
</script>

<template>
  <ArchitectureDiagram
    :arch="arch"
    :service-labels="['Resize', 'Thumbnail', 'CDN', 'Metadata DB']"
    worker-label="Batch logic"
    label="Batch architecture diagram"
  />
</template>
