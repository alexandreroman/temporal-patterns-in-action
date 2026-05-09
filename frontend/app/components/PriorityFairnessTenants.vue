<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import {
  priorityLevel,
  type SimState,
  type Tenant,
} from "~/utils/priority-fairness";

/**
 * Three tenant lanes — each panel has a colored left border, header summary,
 * and a flex-wrap body of priority-tinted ticket chips. The body clips at
 * two rows; if the queue cannot fit, a `+N` chip is rendered inline as the
 * last visible chip with the count of trailing tickets that didn't make it in.
 */

const props = defineProps<{
  tenants: readonly Tenant[];
  state: SimState;
}>();

interface Lane {
  tenant: Tenant;
  chips: { id: string; bg: string; fg: string }[];
  queued: number;
  resolved: number;
}

const lanes = computed<Lane[]>(() =>
  props.tenants.map((tenant) => {
    const queue = props.state.queues[tenant.id] ?? [];
    const chips = queue.map((t) => {
      const lvl = priorityLevel(t.priorityKey);
      return { id: t.id, bg: lvl.bg, fg: lvl.fg };
    });
    return {
      tenant,
      chips,
      queued: queue.length,
      resolved: props.state.resolved[tenant.id] ?? 0,
    };
  }),
);

const root = ref<HTMLElement | null>(null);
// Per-lane visible-chip cap. `undefined` means "show all chips, no `+N`";
// any number ≤ chips.length triggers inline `+N` for the remainder.
const cutoffs = ref<Record<string, number | undefined>>({});

function visibleChips(lane: Lane) {
  const c = cutoffs.value[lane.tenant.id];
  return c === undefined ? lane.chips : lane.chips.slice(0, c);
}

function overflowOf(lane: Lane) {
  const c = cutoffs.value[lane.tenant.id];
  return c === undefined ? 0 : Math.max(0, lane.chips.length - c);
}

// Pad the count with non-breaking spaces so the `+N` text is always at
// least 4 monospace cells wide — matching a 4-digit ticket ID. With
// `tabular-nums + font-mono`, that makes the chip pixel-identical in
// width to a ticket regardless of how big N is.
function overflowText(lane: Lane): string {
  return `+${overflowOf(lane).toString()}`.padStart(4, " ");
}

function adjustLane(laneId: string) {
  const el = root.value?.querySelector<HTMLElement>(
    `[data-wrap="${laneId}"]`,
  );
  if (!el) return;
  if (el.scrollHeight <= el.clientHeight + 1) return;
  const children = Array.from(el.children) as HTMLElement[];
  const limit = el.clientHeight;
  let firstHidden = children.length;
  for (let i = 0; i < children.length; i++) {
    const c = children[i];
    if (c && c.offsetTop + c.offsetHeight > limit + 1) {
      firstHidden = i;
      break;
    }
  }
  // Drop one extra slot for the inline `+N` chip itself.
  cutoffs.value[laneId] = Math.max(0, firstHidden - 1);
}

let measureRaf = 0;
function scheduleMeasure() {
  if (measureRaf) return;
  measureRaf = requestAnimationFrame(() => {
    measureRaf = 0;
    for (const lane of lanes.value) adjustLane(lane.tenant.id);
  });
}

// On resize, the cached cutoff is no longer trustworthy — drop it so the
// next render shows every chip and a fresh measurement can settle.
function scheduleReset() {
  for (const lane of lanes.value) cutoffs.value[lane.tenant.id] = undefined;
  nextTick(scheduleMeasure);
}

let ro: ResizeObserver | null = null;

onMounted(() => {
  scheduleReset();
  if (!root.value) return;
  ro = new ResizeObserver(scheduleReset);
  for (const el of root.value.querySelectorAll<HTMLElement>("[data-wrap]")) {
    ro.observe(el);
  }
});

onUnmounted(() => ro?.disconnect());

watch(lanes, () => scheduleMeasure(), { deep: true });
</script>

<template>
  <div ref="root" class="flex flex-col gap-2">
    <div
      v-for="lane in lanes"
      :key="lane.tenant.id"
      class="rounded-xl border border-l-[3px] border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900"
      :style="{ borderLeftColor: lane.tenant.color }"
    >
      <div class="flex flex-wrap items-baseline justify-between gap-2">
        <div class="flex items-baseline gap-2">
          <span class="text-xs font-medium text-slate-700 dark:text-slate-200">
            {{ lane.tenant.name }}
          </span>
          <span class="text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
            {{ lane.tenant.tier }} &middot; weight {{ lane.tenant.weight }}
          </span>
        </div>
        <div class="font-mono text-[11px] tabular-nums text-slate-500 dark:text-slate-400">
          queued: {{ lane.queued }} &middot; resolved: {{ lane.resolved }}
        </div>
      </div>

      <div
        :data-wrap="lane.tenant.id"
        class="relative mt-2 flex h-[48px] flex-wrap content-start items-start gap-1 overflow-hidden"
      >
        <span
          v-for="t in visibleChips(lane)"
          :key="t.id"
          class="rounded-md px-1.5 py-0.5 font-mono text-[11px] tabular-nums"
          :style="{ backgroundColor: t.bg, color: t.fg }"
        >
          {{ t.id }}
        </span>
        <span
          v-if="overflowOf(lane) > 0"
          class="rounded-md bg-slate-200 px-1.5 py-0.5 font-mono text-[11px] tabular-nums text-slate-600 dark:bg-slate-700 dark:text-slate-300"
          >{{ overflowText(lane) }}</span
        >
        <span
          v-if="lane.chips.length === 0"
          class="font-mono text-[11px] text-slate-400 dark:text-slate-500"
        >
          (empty)
        </span>
      </div>
    </div>
  </div>
</template>
