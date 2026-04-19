<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Architecture diagram for the long-running batch pattern.
 *
 * Nodes: UI -> Temporal -> Worker -> (Resize | Thumbnail | CDN | Metadata DB)
 * The specific downstream service lit up is derived from the `service` field
 * on `batch.item.*` events — the activity name is always `process-image`
 * and carries no routing information on its own.
 */

type NodeState = "idle" | "active" | "ok" | "warn" | "error";
type EdgeState = "idle" | "active" | "warn" | "error";

type NodeId = "nui" | "ntmp" | "nwk" | "svc_resize" | "svc_thumb" | "svc_cdn" | "svc_db";

type EdgeId = "e_ui_tmp" | "e_tmp_wk" | "e_wk_resize" | "e_wk_thumb" | "e_wk_cdn" | "e_wk_db";

type Nodes = Record<NodeId, NodeState>;
type Edges = Record<EdgeId, EdgeState>;

interface ArchState {
  nodes: Nodes;
  edges: Edges;
  running: boolean;
}

const NODE_IDS: NodeId[] = [
  "nui",
  "ntmp",
  "nwk",
  "svc_resize",
  "svc_thumb",
  "svc_cdn",
  "svc_db",
];
const EDGE_IDS: EdgeId[] = [
  "e_ui_tmp",
  "e_tmp_wk",
  "e_wk_resize",
  "e_wk_thumb",
  "e_wk_cdn",
  "e_wk_db",
];

const SERVICE_NODES: NodeId[] = ["svc_resize", "svc_thumb", "svc_cdn", "svc_db"];
const SERVICE_EDGES: EdgeId[] = ["e_wk_resize", "e_wk_thumb", "e_wk_cdn", "e_wk_db"];

const SERVICE_TO_NODE: Record<string, { node: NodeId; edge: EdgeId }> = {
  resize: { node: "svc_resize", edge: "e_wk_resize" },
  thumbnail: { node: "svc_thumb", edge: "e_wk_thumb" },
  cdn: { node: "svc_cdn", edge: "e_wk_cdn" },
  metadata: { node: "svc_db", edge: "e_wk_db" },
};

function initialNodes(): Nodes {
  return {
    nui: "idle",
    ntmp: "idle",
    nwk: "idle",
    svc_resize: "idle",
    svc_thumb: "idle",
    svc_cdn: "idle",
    svc_db: "idle",
  };
}

function initialEdges(): Edges {
  return {
    e_ui_tmp: "idle",
    e_tmp_wk: "idle",
    e_wk_resize: "idle",
    e_wk_thumb: "idle",
    e_wk_cdn: "idle",
    e_wk_db: "idle",
  };
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const arch = computed<ArchState>(() => {
  const nodes: Nodes = initialNodes();
  const edges: Edges = initialEdges();

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
      case "batch.item.started": {
        if (svc) {
          nodes.ntmp = "active";
          nodes.nwk = "active";
          nodes[svc.node] = "active";
          edges[svc.edge] = "active";
        }
        break;
      }

      case "batch.item.completed": {
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
        break;
      }

      case "batch.item.attempt_failed": {
        // Transient — the retry will flip this back to active/ok.
        if (svc) {
          nodes[svc.node] = "warn";
          edges[svc.edge] = "warn";
        }
        break;
      }

      case "batch.summary.reported": {
        // The summary activity writes to the Metadata DB.
        nodes.svc_db = "active";
        edges.e_wk_db = "active";
        break;
      }

      case "progress.workflow.completed":
        resetAll(nodes, edges);
        running = false;
        nodes.ntmp = "ok";
        nodes.nui = "ok";
        break;

      case "progress.workflow.failed":
        resetServices(nodes, edges);
        running = false;
        nodes.nui = "error";
        nodes.ntmp = "error";
        nodes.nwk = "error";
        edges.e_ui_tmp = "error";
        edges.e_tmp_wk = "error";
        break;
    }
  }

  // Keep the UI→Temporal→Worker strip lit while the run is in flight, since
  // no workflow.started event arrives to set it up explicitly.
  if (running) {
    if (nodes.nui === "idle") nodes.nui = "active";
    if (nodes.ntmp === "idle") nodes.ntmp = "active";
    if (nodes.nwk === "idle") nodes.nwk = "active";
    if (edges.e_ui_tmp === "idle") edges.e_ui_tmp = "active";
    if (edges.e_tmp_wk === "idle") edges.e_tmp_wk = "active";
  }

  return { nodes, edges, running };
});

function resetAll(nodes: Nodes, edges: Edges) {
  for (const id of NODE_IDS) nodes[id] = "idle";
  for (const id of EDGE_IDS) edges[id] = "idle";
}

function resetServices(nodes: Nodes, edges: Edges) {
  for (const id of SERVICE_NODES) {
    if (nodes[id] !== "ok" && nodes[id] !== "error") {
      nodes[id] = "idle";
    }
  }
  for (const id of SERVICE_EDGES) {
    if (edges[id] !== "error") edges[id] = "idle";
  }
}

// Style mappings — kept identical to SagaArchitecture to share visual vocabulary.
const nodeFill: Record<NodeState, string> = {
  idle: "fill-slate-100 dark:fill-slate-800",
  active: "fill-blue-100 dark:fill-blue-900",
  ok: "fill-emerald-100 dark:fill-emerald-900",
  warn: "fill-amber-100 dark:fill-amber-900",
  error: "fill-rose-100 dark:fill-rose-900",
};

const nodeStroke: Record<NodeState, string> = {
  idle: "stroke-slate-300 dark:stroke-slate-600",
  active: "stroke-blue-400 dark:stroke-blue-500",
  ok: "stroke-emerald-400 dark:stroke-emerald-500",
  warn: "stroke-amber-400 dark:stroke-amber-500",
  error: "stroke-rose-400 dark:stroke-rose-500",
};

const edgeStroke: Record<EdgeState, string> = {
  idle: "stroke-slate-300 dark:stroke-slate-600",
  active: "stroke-blue-500 dark:stroke-blue-400",
  warn: "stroke-amber-500 dark:stroke-amber-400",
  error: "stroke-rose-500 dark:stroke-rose-400",
};

const edgeAnim: Record<EdgeState, string> = {
  idle: "",
  active: "edge-flow-active",
  warn: "edge-flow-active",
  error: "edge-flow-error",
};
</script>

<template>
  <svg viewBox="0 0 680 240" class="w-full" role="img" aria-label="Batch architecture diagram">
    <!-- UI -->
    <g>
      <rect
        x="40"
        y="95"
        width="100"
        height="50"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.nui], nodeStroke[arch.nodes.nui]]"
        stroke-width="1"
      />
      <text
        x="90"
        y="115"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[13px] font-medium"
      >
        UI
      </text>
      <text
        x="90"
        y="131"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-500 dark:fill-slate-400 text-[11px]"
      >
        Client
      </text>
    </g>

    <!-- Temporal -->
    <g>
      <rect
        x="190"
        y="95"
        width="130"
        height="50"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.ntmp], nodeStroke[arch.nodes.ntmp]]"
        stroke-width="1"
      />
      <text
        x="255"
        y="115"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[13px] font-medium"
      >
        Temporal
      </text>
      <text
        x="255"
        y="131"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-500 dark:fill-slate-400 text-[11px]"
      >
        Orchestrator
      </text>
    </g>

    <!-- Worker -->
    <g>
      <rect
        x="370"
        y="95"
        width="100"
        height="50"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.nwk], nodeStroke[arch.nodes.nwk]]"
        stroke-width="1"
      />
      <text
        x="420"
        y="115"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[13px] font-medium"
      >
        Worker
      </text>
      <text
        x="420"
        y="131"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-500 dark:fill-slate-400 text-[11px]"
      >
        Batch logic
      </text>
    </g>

    <!-- Resize service -->
    <g>
      <rect
        x="530"
        y="12"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_resize], nodeStroke[arch.nodes.svc_resize]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="32"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Resize
      </text>
    </g>

    <!-- Thumbnail service -->
    <g>
      <rect
        x="530"
        y="68"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_thumb], nodeStroke[arch.nodes.svc_thumb]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="88"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Thumbnail
      </text>
    </g>

    <!-- CDN service -->
    <g>
      <rect
        x="530"
        y="124"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_cdn], nodeStroke[arch.nodes.svc_cdn]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="144"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        CDN
      </text>
    </g>

    <!-- Metadata DB -->
    <g>
      <rect
        x="530"
        y="180"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_db], nodeStroke[arch.nodes.svc_db]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="200"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Metadata DB
      </text>
    </g>

    <!-- Edges -->
    <line
      x1="140"
      y1="120"
      x2="188"
      y2="120"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_ui_tmp], arch.running ? edgeAnim[arch.edges.e_ui_tmp] : '']"
      :stroke-width="arch.edges.e_ui_tmp !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.e_ui_tmp !== 'idle' && arch.edges.e_ui_tmp !== 'error'
          ? '6 4'
          : 'none'
      "
    />
    <line
      x1="320"
      y1="120"
      x2="368"
      y2="120"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_tmp_wk], arch.running ? edgeAnim[arch.edges.e_tmp_wk] : '']"
      :stroke-width="arch.edges.e_tmp_wk !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.e_tmp_wk !== 'idle' && arch.edges.e_tmp_wk !== 'error'
          ? '6 4'
          : 'none'
      "
    />
    <line
      x1="470"
      y1="108"
      x2="528"
      y2="38"
      fill="none"
      class="transition-all duration-300"
      :class="[
        edgeStroke[arch.edges.e_wk_resize],
        arch.running ? edgeAnim[arch.edges.e_wk_resize] : '',
      ]"
      :stroke-width="arch.edges.e_wk_resize !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.e_wk_resize !== 'idle' && arch.edges.e_wk_resize !== 'error'
          ? '6 4'
          : 'none'
      "
    />
    <line
      x1="470"
      y1="115"
      x2="528"
      y2="88"
      fill="none"
      class="transition-all duration-300"
      :class="[
        edgeStroke[arch.edges.e_wk_thumb],
        arch.running ? edgeAnim[arch.edges.e_wk_thumb] : '',
      ]"
      :stroke-width="arch.edges.e_wk_thumb !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.e_wk_thumb !== 'idle' && arch.edges.e_wk_thumb !== 'error'
          ? '6 4'
          : 'none'
      "
    />
    <line
      x1="470"
      y1="125"
      x2="528"
      y2="144"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_wk_cdn], arch.running ? edgeAnim[arch.edges.e_wk_cdn] : '']"
      :stroke-width="arch.edges.e_wk_cdn !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.e_wk_cdn !== 'idle' && arch.edges.e_wk_cdn !== 'error'
          ? '6 4'
          : 'none'
      "
    />
    <line
      x1="470"
      y1="132"
      x2="528"
      y2="200"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_wk_db], arch.running ? edgeAnim[arch.edges.e_wk_db] : '']"
      :stroke-width="arch.edges.e_wk_db !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.e_wk_db !== 'idle' && arch.edges.e_wk_db !== 'error'
          ? '6 4'
          : 'none'
      "
    />
  </svg>
</template>
