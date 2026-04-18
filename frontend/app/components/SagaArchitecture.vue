<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";

/**
 * Architecture diagram for the Saga pattern.
 *
 * Nodes: UI -> Temporal -> Worker -> (Inventory | Payment |
 *                                          Shipping | Notification)
 * Edges light up as events arrive from the SSE stream.
 */

type NodeState = "idle" | "active" | "ok" | "warn" | "error";
type EdgeState = "idle" | "active" | "warn" | "error";

const NODE_IDS = [
  "nui",
  "ntmp",
  "nwk",
  "svc_inventory",
  "svc_payment",
  "svc_shipping",
  "svc_notif",
] as const;
type NodeId = (typeof NODE_IDS)[number];

const EDGE_IDS = [
  "e_ui_tmp",
  "e_tmp_wk",
  "e_wk_inventory",
  "e_wk_payment",
  "e_wk_shipping",
  "e_wk_notif",
] as const;
type EdgeId = (typeof EDGE_IDS)[number];

type Nodes = Record<NodeId, NodeState>;
type Edges = Record<EdgeId, EdgeState>;

interface ArchState {
  nodes: Nodes;
  edges: Edges;
  running: boolean;
}

const SERVICE_NODES: NodeId[] = ["svc_inventory", "svc_payment", "svc_shipping", "svc_notif"];
const SERVICE_EDGES: EdgeId[] = ["e_wk_inventory", "e_wk_payment", "e_wk_shipping", "e_wk_notif"];

const STEP_TO_SVC: Record<string, { node: NodeId; edge: EdgeId }> = {
  "reserve-inventory": { node: "svc_inventory", edge: "e_wk_inventory" },
  "release-inventory": { node: "svc_inventory", edge: "e_wk_inventory" },
  "charge-payment": { node: "svc_payment", edge: "e_wk_payment" },
  "refund-payment": { node: "svc_payment", edge: "e_wk_payment" },
  "ship-order": { node: "svc_shipping", edge: "e_wk_shipping" },
  "cancel-shipment": { node: "svc_shipping", edge: "e_wk_shipping" },
  "send-confirmation": { node: "svc_notif", edge: "e_wk_notif" },
  "retract-email": { node: "svc_notif", edge: "e_wk_notif" },
};

function initialNodes(): Nodes {
  return {
    nui: "idle",
    ntmp: "idle",
    nwk: "idle",
    svc_inventory: "idle",
    svc_payment: "idle",
    svc_shipping: "idle",
    svc_notif: "idle",
  };
}

function initialEdges(): Edges {
  return {
    e_ui_tmp: "idle",
    e_tmp_wk: "idle",
    e_wk_inventory: "idle",
    e_wk_payment: "idle",
    e_wk_shipping: "idle",
    e_wk_notif: "idle",
  };
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const arch = computed<ArchState>(() => {
  const nodes: Nodes = initialNodes();
  const edges: Edges = initialEdges();

  let compensating = false;
  // Running stays true as long as we've seen at least one event and no
  // terminal event (completed/failed). NATS subscription races the workflow
  // start, so `progress.workflow.started` is usually lost — we cannot rely
  // on it as the only signal to turn animations on.
  let running = props.events.length > 0;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;

    switch (env.type) {
      case "progress.workflow.started":
        resetAll(nodes, edges);
        running = true;
        nodes.nui = "active";
        edges.e_ui_tmp = "active";
        nodes.ntmp = "active";
        edges.e_tmp_wk = "active";
        nodes.nwk = "active";
        break;

      case "progress.step.started": {
        const step = String(data.step ?? "");
        const svc = STEP_TO_SVC[step];
        if (svc) {
          resetServices(nodes, edges);
          nodes.ntmp = "active";
          nodes.nwk = compensating ? "warn" : "active";
          edges[svc.edge] = compensating ? "warn" : "active";
          nodes[svc.node] = compensating ? "warn" : "active";
        }
        break;
      }

      case "progress.step.completed": {
        const step = String(data.step ?? "");
        const svc = STEP_TO_SVC[step];
        if (svc) {
          nodes[svc.node] = "ok";
          edges[svc.edge] = "idle";
        }
        break;
      }

      case "progress.step.failed": {
        const step = String(data.step ?? "");
        const svc = STEP_TO_SVC[step];
        if (svc) {
          nodes[svc.node] = "error";
          edges[svc.edge] = "error";
          nodes.nwk = "error";
        }
        break;
      }

      case "progress.compensation.started":
        compensating = true;
        nodes.ntmp = "warn";
        nodes.nwk = "warn";
        break;

      case "progress.compensation.completed":
        compensating = false;
        break;

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

// Style mappings
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

// Animated flow along the dashed stroke when an edge is active —
// gives the impression of data moving between the two nodes.
const edgeAnim: Record<EdgeState, string> = {
  idle: "",
  active: "edge-flow-active",
  warn: "edge-flow-warn",
  error: "edge-flow-error",
};
</script>

<template>
  <svg viewBox="0 0 680 240" class="w-full" role="img" aria-label="Saga architecture diagram">
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
        Saga logic
      </text>
    </g>

    <!-- Inventory service -->
    <g>
      <rect
        x="530"
        y="12"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_inventory], nodeStroke[arch.nodes.svc_inventory]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="32"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Inventory
      </text>
    </g>

    <!-- Payment service -->
    <g>
      <rect
        x="530"
        y="68"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_payment], nodeStroke[arch.nodes.svc_payment]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="88"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Payment
      </text>
    </g>

    <!-- Shipping service -->
    <g>
      <rect
        x="530"
        y="124"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_shipping], nodeStroke[arch.nodes.svc_shipping]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="144"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Shipping
      </text>
    </g>

    <!-- Notification service -->
    <g>
      <rect
        x="530"
        y="180"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.svc_notif], nodeStroke[arch.nodes.svc_notif]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="200"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Notification
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
      :stroke-dasharray="arch.running && arch.edges.e_ui_tmp !== 'idle' && arch.edges.e_ui_tmp !== 'error' ? '6 4' : 'none'"
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
      :stroke-dasharray="arch.running && arch.edges.e_tmp_wk !== 'idle' && arch.edges.e_tmp_wk !== 'error' ? '6 4' : 'none'"
    />
    <line
      x1="470"
      y1="108"
      x2="528"
      y2="38"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_wk_inventory], arch.running ? edgeAnim[arch.edges.e_wk_inventory] : '']"
      :stroke-width="arch.edges.e_wk_inventory !== 'idle' ? 3 : 2"
      :stroke-dasharray="arch.running && arch.edges.e_wk_inventory !== 'idle' && arch.edges.e_wk_inventory !== 'error' ? '6 4' : 'none'"
    />
    <line
      x1="470"
      y1="115"
      x2="528"
      y2="88"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_wk_payment], arch.running ? edgeAnim[arch.edges.e_wk_payment] : '']"
      :stroke-width="arch.edges.e_wk_payment !== 'idle' ? 3 : 2"
      :stroke-dasharray="arch.running && arch.edges.e_wk_payment !== 'idle' && arch.edges.e_wk_payment !== 'error' ? '6 4' : 'none'"
    />
    <line
      x1="470"
      y1="125"
      x2="528"
      y2="144"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_wk_shipping], arch.running ? edgeAnim[arch.edges.e_wk_shipping] : '']"
      :stroke-width="arch.edges.e_wk_shipping !== 'idle' ? 3 : 2"
      :stroke-dasharray="arch.running && arch.edges.e_wk_shipping !== 'idle' && arch.edges.e_wk_shipping !== 'error' ? '6 4' : 'none'"
    />
    <line
      x1="470"
      y1="132"
      x2="528"
      y2="200"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.e_wk_notif], arch.running ? edgeAnim[arch.edges.e_wk_notif] : '']"
      :stroke-width="arch.edges.e_wk_notif !== 'idle' ? 3 : 2"
      :stroke-dasharray="arch.running && arch.edges.e_wk_notif !== 'idle' && arch.edges.e_wk_notif !== 'error' ? '6 4' : 'none'"
    />
  </svg>
</template>
