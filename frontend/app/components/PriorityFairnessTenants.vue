<script setup lang="ts">
import { computed } from "vue";
import {
  priorityLevel,
  TENANT_QUEUE_VISIBLE,
  type SimState,
  type Tenant,
} from "~/utils/priority-fairness";

/**
 * Three tenant lanes — each panel has a colored left border, header summary,
 * and a flex-wrap body of priority-tinted ticket chips. The body caps at
 * TENANT_QUEUE_VISIBLE chips and shows a `+N` overflow tile when the queue
 * is deeper.
 */

const props = defineProps<{
  tenants: readonly Tenant[];
  state: SimState;
}>();

interface Lane {
  tenant: Tenant;
  visible: { id: string; bg: string; fg: string }[];
  overflow: number;
  queued: number;
  resolved: number;
}

const lanes = computed<Lane[]>(() =>
  props.tenants.map((tenant) => {
    const queue = props.state.queues[tenant.id] ?? [];
    const visible = queue.slice(0, TENANT_QUEUE_VISIBLE).map((t) => {
      const lvl = priorityLevel(t.priorityKey);
      return { id: t.id, bg: lvl.bg, fg: lvl.fg };
    });
    return {
      tenant,
      visible,
      overflow: Math.max(0, queue.length - TENANT_QUEUE_VISIBLE),
      queued: queue.length,
      resolved: props.state.resolved[tenant.id] ?? 0,
    };
  }),
);
</script>

<template>
  <div class="flex flex-col gap-2">
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

      <div class="mt-2 flex min-h-[26px] flex-wrap gap-1">
        <span
          v-for="t in lane.visible"
          :key="t.id"
          class="rounded-md px-1.5 py-0.5 font-mono text-[11px] tabular-nums"
          :style="{ backgroundColor: t.bg, color: t.fg }"
        >
          {{ t.id }}
        </span>
        <span
          v-if="lane.overflow > 0"
          class="rounded-md border border-slate-300 bg-slate-100 px-1.5 py-0.5 font-mono text-[11px] tabular-nums text-slate-600 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300"
        >
          +{{ lane.overflow }}
        </span>
        <span
          v-if="lane.visible.length === 0 && lane.overflow === 0"
          class="font-mono text-[11px] text-slate-400 dark:text-slate-500"
        >
          (empty)
        </span>
      </div>
    </div>
  </div>
</template>
