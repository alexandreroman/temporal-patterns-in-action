<script setup lang="ts">
import type { ArchState, EdgeState, NodeState } from "~/types/architecture";

defineProps<{
  arch: ArchState;
  serviceLabels: [string, string, string, string];
  workerLabel: string;
  label: string;
  codec?: string;
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
        x="370"
        y="75"
        width="100"
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

    <!-- Service 1 -->
    <g>
      <rect
        x="530"
        y="8"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.s1], nodeStroke[arch.nodes.s1]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="28"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[11px] font-medium"
      >
        {{ serviceLabels[0] }}
      </text>
    </g>

    <!-- Service 2 -->
    <g>
      <rect
        x="530"
        y="56"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.s2], nodeStroke[arch.nodes.s2]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="76"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[11px] font-medium"
      >
        {{ serviceLabels[1] }}
      </text>
    </g>

    <!-- Service 3 -->
    <g>
      <rect
        x="530"
        y="104"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.s3], nodeStroke[arch.nodes.s3]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="124"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[11px] font-medium"
      >
        {{ serviceLabels[2] }}
      </text>
    </g>

    <!-- Service 4 -->
    <g>
      <rect
        x="530"
        y="152"
        width="120"
        height="40"
        rx="8"
        class="transition-all duration-300"
        :class="[nodeFill[arch.nodes.s4], nodeStroke[arch.nodes.s4]]"
        stroke-width="1"
      />
      <text
        x="590"
        y="172"
        text-anchor="middle"
        dominant-baseline="central"
        class="fill-slate-800 dark:fill-slate-100 text-[11px] font-medium"
      >
        {{ serviceLabels[3] }}
      </text>
    </g>

    <!-- Edges -->
    <line
      x1="140"
      y1="100"
      x2="188"
      y2="100"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.ui_tmp], arch.running ? edgeAnim[arch.edges.ui_tmp] : '']"
      :stroke-width="arch.edges.ui_tmp !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.ui_tmp !== 'idle' && arch.edges.ui_tmp !== 'error'
          ? '6 4'
          : 'none'
      "
    />
    <line
      x1="320"
      y1="100"
      x2="368"
      y2="100"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.tmp_wk], arch.running ? edgeAnim[arch.edges.tmp_wk] : '']"
      :stroke-width="arch.edges.tmp_wk !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.tmp_wk !== 'idle' && arch.edges.tmp_wk !== 'error'
          ? '6 4'
          : 'none'
      "
    />
    <line
      x1="470"
      y1="88"
      x2="528"
      y2="28"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.wk_s1], arch.running ? edgeAnim[arch.edges.wk_s1] : '']"
      :stroke-width="arch.edges.wk_s1 !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.wk_s1 !== 'idle' && arch.edges.wk_s1 !== 'error' ? '6 4' : 'none'
      "
    />
    <line
      x1="470"
      y1="95"
      x2="528"
      y2="76"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.wk_s2], arch.running ? edgeAnim[arch.edges.wk_s2] : '']"
      :stroke-width="arch.edges.wk_s2 !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.wk_s2 !== 'idle' && arch.edges.wk_s2 !== 'error' ? '6 4' : 'none'
      "
    />
    <line
      x1="470"
      y1="105"
      x2="528"
      y2="124"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.wk_s3], arch.running ? edgeAnim[arch.edges.wk_s3] : '']"
      :stroke-width="arch.edges.wk_s3 !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.wk_s3 !== 'idle' && arch.edges.wk_s3 !== 'error' ? '6 4' : 'none'
      "
    />
    <line
      x1="470"
      y1="112"
      x2="528"
      y2="172"
      fill="none"
      class="transition-all duration-300"
      :class="[edgeStroke[arch.edges.wk_s4], arch.running ? edgeAnim[arch.edges.wk_s4] : '']"
      :stroke-width="arch.edges.wk_s4 !== 'idle' ? 3 : 2"
      :stroke-dasharray="
        arch.running && arch.edges.wk_s4 !== 'idle' && arch.edges.wk_s4 !== 'error' ? '6 4' : 'none'
      "
    />
  </svg>
</template>
