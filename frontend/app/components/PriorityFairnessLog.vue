<script setup lang="ts">
import { computed } from "vue";
import { formatLogTime, priorityLevel, tenantById, type LogEntry } from "~/utils/priority-fairness";

/**
 * Resolution log — newest first, monospace. Each row shows the relative
 * timestamp (MM:SS.d), the priority label chip, the ticket as a
 * priority-colored chip (matching the queue chips), a tenant-colored pill,
 * and the agent slot that resolved it.
 */

const props = defineProps<{
  log: LogEntry[];
  startTime: number;
}>();

interface Row {
  key: string;
  time: string;
  priorityLabel: string;
  priorityBg: string;
  priorityFg: string;
  tenantName: string;
  tenantColor: string;
  ticketId: string;
  agent: string;
}

const rows = computed<Row[]>(() =>
  props.log.map((entry, idx) => {
    const lvl = priorityLevel(entry.priorityKey);
    const tenant = tenantById(entry.tenantId);
    return {
      // Use index plus ticket so identical tickets across resets keep distinct keys.
      key: `${idx}-${entry.ticket}`,
      time: formatLogTime(entry.time - props.startTime),
      priorityLabel: lvl.label,
      priorityBg: lvl.bg,
      priorityFg: lvl.fg,
      tenantName: tenant.name,
      tenantColor: tenant.color,
      ticketId: entry.ticket,
      agent: entry.agent,
    };
  }),
);
</script>

<template>
  <div
    class="flex h-full flex-col rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900"
  >
    <div
      class="flex items-center justify-between gap-3 border-b border-slate-200 px-3 py-2 dark:border-slate-700"
    >
      <div class="text-xs font-medium text-slate-700 dark:text-slate-300">
        Resolution log <span class="text-slate-400 dark:text-slate-500">&middot; newest first</span>
      </div>
      <div class="font-mono text-[11px] tabular-nums text-slate-500 dark:text-slate-400">
        {{ log.length }}
      </div>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto px-3 py-2">
      <p
        v-if="rows.length === 0"
        class="py-6 text-center text-[11px] text-slate-400 dark:text-slate-500"
      >
        (no resolutions yet)
      </p>
      <ul v-else class="flex flex-col gap-1 font-mono text-[11px]">
        <li v-for="row in rows" :key="row.key" class="flex items-center gap-2">
          <span class="tabular-nums text-slate-500 dark:text-slate-400">{{ row.time }}</span>
          <span
            class="rounded-md px-1.5 py-0.5 tabular-nums"
            :style="{ backgroundColor: row.priorityBg, color: row.priorityFg }"
          >
            {{ row.priorityLabel }}
          </span>
          <span
            class="rounded-md px-1.5 py-0.5 tabular-nums"
            :style="{ backgroundColor: row.priorityBg, color: row.priorityFg }"
          >
            {{ row.ticketId }}
          </span>
          <span
            class="min-w-0 flex-1 truncate rounded-md px-1.5 py-0.5 text-white"
            :style="{ backgroundColor: row.tenantColor }"
            :title="row.tenantName"
          >
            {{ row.tenantName }}
          </span>
          <span class="tabular-nums text-slate-500 dark:text-slate-400">
            {{ row.agent }}
          </span>
        </li>
      </ul>
    </div>
  </div>
</template>
