<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { ArchState, EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Batch architecture: UI -> Temporal -> Worker -> (Resize | Thumbnail | CDN |
 *                                                      Metadata DB)
 * The active service is derived from the `service` field on `batch.item.*`
 * events — the activity name is always `process-image` and carries no routing
 * information on its own.
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

const arch = computed<ArchState>(() => {
  const nodes = initialNodes();
  const edges = initialEdges();

  // Running stays true as long as we've seen at least one event and no
  // terminal event has closed the run. The worker no longer emits a
  // workflow.started signal — the Nuxt SSE endpoint synthesises only
  // the terminal events — so the first observed event anchors the run.
  let running = props.events.length > 0;

  for (const env of props.events) {
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
    :service-labels="['Resize', 'Thumbnail', 'CDN', 'Metadata DB']"
    worker-label="Batch logic"
    label="Batch architecture diagram"
  />
</template>
