<script setup lang="ts">
import { computed } from "vue";
import {
  HISTORY_LEN,
  type HistorySample,
  type Tenant,
  type TenantId,
} from "~/utils/priority-fairness";

/**
 * Throughput chart — three smooth lines (one per tenant) plotting
 * resolutions per second over the rolling HISTORY_LEN window. Uses a fixed
 * SVG viewBox + preserveAspectRatio="none" so the lines stretch to fill the
 * container; vector-effect="non-scaling-stroke" keeps stroke widths
 * consistent regardless of horizontal scale.
 */

const VIEW_W = 800;
const VIEW_H = 130;
const PAD_X = 4;
const PAD_TOP = 8;
const PAD_BOTTOM = 18;
const RES_PER_SEC_WINDOW = 4; // 1 second @ 250 ms ticks

const props = defineProps<{
  history: HistorySample[];
  tenants: readonly Tenant[];
}>();

interface Series {
  id: TenantId;
  name: string;
  color: string;
  weight: number;
  points: string;
}

const yMax = computed(() => {
  let max = 4;
  for (const series of seriesPoints.value) {
    for (const v of series.values) if (v > max) max = v;
  }
  return max;
});

interface RawSeries {
  id: TenantId;
  name: string;
  color: string;
  weight: number;
  values: number[];
}

const seriesPoints = computed<RawSeries[]>(() => {
  const samples = props.history;
  const len = samples.length;
  return props.tenants.map((tenant) => {
    const values: number[] = [];
    for (let i = 0; i < len; i++) {
      // Sum the deltas over the last RES_PER_SEC_WINDOW samples to get a
      // rolling resolutions-per-second rate.
      let acc = 0;
      const start = Math.max(0, i - RES_PER_SEC_WINDOW + 1);
      for (let j = start; j <= i; j++) {
        acc += samples[j]?.[tenant.id] ?? 0;
      }
      values.push(acc);
    }
    return {
      id: tenant.id,
      name: tenant.name,
      color: tenant.color,
      weight: tenant.weight,
      values,
    };
  });
});

const series = computed<Series[]>(() => {
  const max = yMax.value;
  const innerW = VIEW_W - PAD_X * 2;
  const innerH = VIEW_H - PAD_TOP - PAD_BOTTOM;
  // The X axis spans the full HISTORY_LEN even when fewer samples exist;
  // partial history grows from the left. This keeps the chart anchored.
  const stepX = innerW / Math.max(1, HISTORY_LEN - 1);

  return seriesPoints.value.map((raw) => {
    const points = raw.values
      .map((v, i) => {
        const x = PAD_X + i * stepX;
        const y = PAD_TOP + innerH - (v / max) * innerH;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(" ");
    return {
      id: raw.id,
      name: raw.name,
      color: raw.color,
      weight: raw.weight,
      points,
    };
  });
});

// Two horizontal gridlines: midpoint and top of the inner area.
const gridlines = computed(() => {
  const innerH = VIEW_H - PAD_TOP - PAD_BOTTOM;
  return [PAD_TOP + innerH, PAD_TOP + innerH * 0.5, PAD_TOP];
});
</script>

<template>
  <div
    class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900"
  >
    <div class="flex items-center justify-between gap-3">
      <div class="text-xs font-medium text-slate-700 dark:text-slate-300">
        throughput
        <span class="text-slate-400 dark:text-slate-500">
          &middot; resolutions/sec, last 20 s
        </span>
      </div>
      <div class="font-mono text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        max {{ yMax }}
      </div>
    </div>

    <svg
      :viewBox="`0 0 ${VIEW_W} ${VIEW_H}`"
      preserveAspectRatio="none"
      class="mt-2 block h-[130px] w-full"
      role="img"
      aria-label="Throughput chart per tenant"
    >
      <line
        v-for="(y, idx) in gridlines"
        :key="idx"
        :x1="PAD_X"
        :x2="VIEW_W - PAD_X"
        :y1="y"
        :y2="y"
        stroke="currentColor"
        stroke-width="1"
        stroke-dasharray="2 4"
        vector-effect="non-scaling-stroke"
        class="text-slate-300 dark:text-slate-700"
      />
      <polyline
        v-for="s in series"
        :key="s.id"
        :points="s.points"
        fill="none"
        :stroke="s.color"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        vector-effect="non-scaling-stroke"
      />
    </svg>

    <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px]">
      <div v-for="s in series" :key="s.id" class="flex items-center gap-1.5">
        <span class="inline-block h-2 w-2 rounded-full" :style="{ backgroundColor: s.color }" />
        <span class="text-slate-700 dark:text-slate-200">{{ s.name }}</span>
        <span class="font-mono text-slate-500 dark:text-slate-400">weight={{ s.weight }}</span>
      </div>
    </div>
  </div>
</template>
