<script setup lang="ts">
import type { ArchState, EdgeKey, EdgeState, NodeState } from "~/types/architecture";

const props = defineProps<{
  arch: ArchState;
  serviceLabels: [string, string, string, string];
  workerLabel: string;
  label: string;
  codec?: string;
  caption?: string;
}>();

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
  warn: "edge-flow-active",
  error: "edge-flow-error",
};

// Service slots stacked on the right; `label` indexes the serviceLabels prop.
const SERVICES = [
  { key: "s1", label: 0, y: 8 },
  { key: "s2", label: 1, y: 56 },
  { key: "s3", label: 2, y: 104 },
  { key: "s4", label: 3, y: 152 },
] as const;

const EDGES: ReadonlyArray<{ key: EdgeKey; x1: number; y1: number; x2: number; y2: number }> = [
  { key: "ui_tmp", x1: 140, y1: 100, x2: 188, y2: 100 },
  { key: "tmp_wk", x1: 320, y1: 100, x2: 353, y2: 100 },
  { key: "wk_s1", x1: 485, y1: 88, x2: 528, y2: 28 },
  { key: "wk_s2", x1: 485, y1: 95, x2: 528, y2: 76 },
  { key: "wk_s3", x1: 485, y1: 105, x2: 528, y2: 124 },
  { key: "wk_s4", x1: 485, y1: 112, x2: 528, y2: 172 },
];

function edgeClass(state: EdgeState): string[] {
  return [edgeStroke[state], props.arch.running ? edgeAnim[state] : ""];
}

// Dashes only while data flows, so a settled or failed edge reads as solid.
function edgeDash(state: EdgeState): string {
  return props.arch.running && state !== "idle" && state !== "error" ? "6 4" : "none";
}
</script>

<template>
  <svg viewBox="0 0 680 200" class="w-full" role="img" :aria-label="label">
    <!-- UI -->
    <g>
      <rect
        x="40"
        y="75"
        width="100"
        height="50"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.ui], nodeStroke[arch.nodes.ui]]"
        stroke-width="1"
      />
      <text
        x="90"
        y="95"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        UI
      </text>
      <text
        x="90"
        y="111"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-500 dark:fill-slate-400 text-[10px]"
      >
        Client
      </text>
    </g>

    <!-- Temporal -->
    <g>
      <rect
        x="190"
        y="75"
        width="130"
        height="50"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.temporal], nodeStroke[arch.nodes.temporal]]"
        stroke-width="1"
      />
      <text
        x="255"
        y="95"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Temporal
      </text>
      <text
        x="255"
        y="111"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-500 dark:fill-slate-400 text-[10px]"
      >
        Orchestrator
      </text>
    </g>

    <!-- Worker -->
    <g>
      <rect
        x="355"
        y="75"
        width="130"
        height="50"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.worker], nodeStroke[arch.nodes.worker]]"
        stroke-width="1"
      />
      <text
        x="420"
        y="95"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[12px] font-medium"
      >
        Worker
      </text>
      <text
        x="420"
        y="111"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-500 dark:fill-slate-400 text-[10px]"
      >
        {{ workerLabel }}
      </text>
    </g>

    <!-- Same codec on UI client and worker — Temporal sees ciphertext only. -->
    <template v-if="codec">
      <g v-for="cx in [90, 420]" :key="cx">
        <text
          :x="cx"
          y="28"
          text-anchor="middle"
          dominant-baseline="central"
          class="fill-emerald-500 dark:fill-emerald-400 text-[10px] font-semibold"
        >
          PayloadCodec
        </text>
        <rect
          :x="cx - 50"
          y="38"
          width="100"
          height="22"
          rx="11"
          class="fill-emerald-50 stroke-emerald-400 dark:fill-emerald-950 dark:stroke-emerald-500"
          stroke-width="1"
        />
        <text
          :x="cx"
          y="50"
          text-anchor="middle"
          dominant-baseline="central"
          class="fill-emerald-700 dark:fill-emerald-200 text-[11px] font-medium"
        >
          {{ codec }}
        </text>
        <line
          :x1="cx"
          y1="60"
          :x2="cx"
          y2="75"
          class="stroke-emerald-400 dark:stroke-emerald-500"
          :class="arch.running ? 'edge-flow-active' : ''"
          stroke-width="2"
          stroke-dasharray="3 3"
        />
      </g>
    </template>

    <!-- Services -->
    <g v-for="service in SERVICES" :key="service.key">
      <rect
        x="530"
        :y="service.y"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes[service.key]], nodeStroke[arch.nodes[service.key]]]"
        stroke-width="1"
      />
      <text
        x="590"
        :y="service.y + 20"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[11px] font-medium"
      >
        {{ serviceLabels[service.label] }}
      </text>
    </g>

    <!-- Bottom-left caption -->
    <text
      v-if="caption"
      x="10"
      y="194"
      text-anchor="start"
      class="fill-slate-500 dark:fill-slate-400 text-[11px]"
    >
      {{ caption }}
    </text>

    <!-- Edges -->
    <line
      v-for="edge in EDGES"
      :key="edge.key"
      :x1="edge.x1"
      :y1="edge.y1"
      :x2="edge.x2"
      :y2="edge.y2"
      fill="none"
      class="transition-all duration-300"
      :class="edgeClass(arch.edges[edge.key])"
      :stroke-width="arch.edges[edge.key] !== 'idle' ? 3 : 2"
      :stroke-dasharray="edgeDash(arch.edges[edge.key])"
    />
  </svg>
</template>
