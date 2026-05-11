<script setup lang="ts">
import { computed } from "vue";
import { priorityLevel, tenantById, type Agent } from "~/utils/priority-fairness";

/**
 * Four-card worker pool. Each card shows the slot label and either an idle
 * marker or the in-flight ticket: priority chip, ID, tenant name, and a
 * progress bar tinted in the tenant color.
 */

const props = defineProps<{
  agents: Agent[];
}>();

interface Card {
  slot: string;
  busy: boolean;
  ticketId: string;
  tenantName: string;
  tenantColor: string;
  priorityLabel: string;
  priorityBg: string;
  priorityFg: string;
  pct: number;
}

const cards = computed<Card[]>(() =>
  props.agents.map((agent) => {
    if (!agent.ticket || !agent.tenant) {
      return {
        slot: agent.slot,
        busy: false,
        ticketId: "",
        tenantName: "",
        tenantColor: "",
        priorityLabel: "",
        priorityBg: "",
        priorityFg: "",
        pct: 0,
      };
    }
    const tenant = tenantById(agent.tenant);
    const lvl = priorityLevel(agent.ticket.priority);
    const pct = agent.duration === 0 ? 0 : Math.min(100, (agent.progress / agent.duration) * 100);
    return {
      slot: agent.slot,
      busy: true,
      ticketId: agent.ticket.id,
      tenantName: tenant.name,
      tenantColor: tenant.color,
      priorityLabel: lvl.label,
      priorityBg: lvl.bg,
      priorityFg: lvl.fg,
      pct,
    };
  }),
);
</script>

<template>
  <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
    <div
      v-for="card in cards"
      :key="card.slot"
      class="flex flex-col gap-1.5 rounded-md border border-slate-200 bg-white p-2 dark:border-slate-700 dark:bg-slate-900"
    >
      <div class="flex items-center justify-between">
        <span
          class="font-mono text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400"
        >
          {{ card.slot }}
        </span>
        <span
          class="rounded-md px-1.5 py-0.5 font-mono text-[10px] tabular-nums"
          :class="{ invisible: !card.busy }"
          :style="
            card.busy ? { backgroundColor: card.priorityBg, color: card.priorityFg } : undefined
          "
          aria-hidden="true"
        >
          {{ card.busy ? card.priorityLabel : "P0" }}
        </span>
      </div>

      <div class="flex flex-col gap-1">
        <div class="flex items-baseline justify-between gap-2">
          <span
            v-if="card.busy"
            class="font-mono text-[12px] tabular-nums text-slate-800 dark:text-slate-100"
          >
            {{ card.ticketId }}
          </span>
          <span
            v-else
            class="font-mono text-[12px] tabular-nums text-slate-400 dark:text-slate-500"
          >
            idle
          </span>
          <span
            class="truncate text-[11px] text-slate-500 dark:text-slate-400"
            :class="{ invisible: !card.busy }"
            aria-hidden="true"
          >
            {{ card.busy ? card.tenantName : "—" }}
          </span>
        </div>
        <div class="h-1.5 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
          <div
            v-if="card.busy"
            class="h-full rounded-full transition-all duration-200 ease-linear"
            :style="{ width: `${card.pct}%`, backgroundColor: card.tenantColor }"
          />
        </div>
      </div>
    </div>
  </div>
</template>
