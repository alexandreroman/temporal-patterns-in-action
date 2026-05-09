<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { ArchState, EdgeKey, NodeKey } from "~/types/architecture";

/**
 * Priority and Fairness architecture: UI -> Temporal -> Helpdesk dispatcher ->
 * (Agent a1 | Agent a2 | Agent a3 | Agent a4). Each `helpdesk.ticket.assigned`
 * event lights up the slot of the receiving agent; `helpdesk.ticket.resolved`
 * flashes it ok and the next assignment flips it back to active.
 */

const AGENT_TO_SLOT: Record<string, { node: NodeKey; edge: EdgeKey }> = {
  a1: { node: "s1", edge: "wk_s1" },
  a2: { node: "s2", edge: "wk_s2" },
  a3: { node: "s3", edge: "wk_s3" },
  a4: { node: "s4", edge: "wk_s4" },
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
    const agent = typeof data.agent === "string" ? data.agent : "";
    const svc = AGENT_TO_SLOT[agent];

    switch (env.type) {
      case "helpdesk.ticket.assigned":
        if (svc) {
          nodes.temporal = "active";
          nodes.worker = "active";
          nodes[svc.node] = "active";
          edges[svc.edge] = "active";
        }
        break;

      case "helpdesk.ticket.resolved":
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
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
    :service-labels="['Agent a1', 'Agent a2', 'Agent a3', 'Agent a4']"
    worker-label="Helpdesk dispatcher"
    label="Priority and Fairness architecture diagram"
  />
</template>
