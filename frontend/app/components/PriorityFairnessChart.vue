<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  AGENT_SLOTS,
  HISTORY_LEN,
  TICK_MS,
  priorityLevel,
  tenantById,
  type AgentSlot,
  type Tenant,
  type TicketSpan,
} from "~/utils/priority-fairness";

/**
 * Resolution swim-lane — one horizontal lane per agent. Ticket spans render
 * as colored blocks (color = tenant) flowing right-to-left over the last
 * 20 s; in-flight blocks extend to "now" and advance smoothly via rAF.
 */

const WINDOW_MS = HISTORY_LEN * TICK_MS;

const props = defineProps<{
  spans: TicketSpan[];
  tenants: readonly Tenant[];
  running: boolean;
}>();

const emit = defineEmits<{
  (e: "dump-acme" | "inject-incident"): void;
}>();

interface Block {
  key: string;
  leftPct: number;
  widthPct: number;
  color: string;
  inFlight: boolean;
  title: string;
}

interface Lane {
  slot: AgentSlot;
  blocks: Block[];
}

const now = ref(Date.now());
let raf = 0;
function tick(): void {
  now.value = Date.now();
  raf = requestAnimationFrame(tick);
}
onMounted(() => {
  watch(
    () => props.running,
    (isRunning) => {
      if (isRunning) {
        now.value = Date.now();
        raf = requestAnimationFrame(tick);
      } else if (raf) {
        cancelAnimationFrame(raf);
        raf = 0;
      }
    },
    { immediate: true },
  );
});
onBeforeUnmount(() => {
  if (raf) cancelAnimationFrame(raf);
});

const lanes = computed<Lane[]>(() => {
  const windowEnd = now.value;
  const windowStart = windowEnd - WINDOW_MS;
  const bySlot = new Map<AgentSlot, Block[]>();
  for (const slot of AGENT_SLOTS) bySlot.set(slot, []);

  const ordered = [...props.spans].sort((a, b) => a.startTime - b.startTime);
  for (const span of ordered) {
    if (span.endTime !== null && span.endTime < windowStart) continue;
    const bucket = bySlot.get(span.agent);
    if (!bucket) continue;
    const left = Math.max(span.startTime, windowStart);
    const rightRaw = span.endTime ?? windowEnd;
    const right = Math.min(rightRaw, windowEnd);
    if (right <= left) continue;
    const leftPct = ((left - windowStart) / WINDOW_MS) * 100;
    const widthPct = Math.max(((right - left) / WINDOW_MS) * 100, 0.5);
    const tenant = tenantById(span.tenantId);
    const label = priorityLevel(span.priorityKey).label;
    bucket.push({
      key: `${span.ticketId}-${span.startTime}`,
      leftPct,
      widthPct,
      color: tenant.color,
      inFlight: span.endTime === null,
      title: `${span.ticketId} · ${tenant.name} · ${label}`,
    });
  }

  return AGENT_SLOTS.map((slot) => ({ slot, blocks: bySlot.get(slot) ?? [] }));
});

const inFlightCount = computed(() =>
  props.spans.reduce((n, s) => (s.endTime === null ? n + 1 : n), 0),
);

const FLASH_MS = 700;
const flashDump = ref(false);
const flashIncident = ref(false);
let dumpTimer: ReturnType<typeof setTimeout> | null = null;
let incidentTimer: ReturnType<typeof setTimeout> | null = null;

function clickDump(): void {
  if (!props.running) return;
  emit("dump-acme");
  flashDump.value = true;
  if (dumpTimer) clearTimeout(dumpTimer);
  dumpTimer = setTimeout(() => {
    flashDump.value = false;
    dumpTimer = null;
  }, FLASH_MS);
}

function clickIncident(): void {
  if (!props.running) return;
  emit("inject-incident");
  flashIncident.value = true;
  if (incidentTimer) clearTimeout(incidentTimer);
  incidentTimer = setTimeout(() => {
    flashIncident.value = false;
    incidentTimer = null;
  }, FLASH_MS);
}

onBeforeUnmount(() => {
  if (dumpTimer) clearTimeout(dumpTimer);
  if (incidentTimer) clearTimeout(incidentTimer);
});
</script>

<template>
  <div
    class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900"
  >
    <div class="flex items-center justify-between gap-3">
      <div class="text-xs font-medium text-slate-700 dark:text-slate-300">
        Resolved per agent
        <span class="text-slate-400 dark:text-slate-500"> &middot; last 20 s, color = tenant </span>
      </div>
      <div class="font-mono text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        in flight {{ inFlightCount }}
      </div>
    </div>

    <div class="mt-2 flex flex-col gap-1.5">
      <div v-for="lane in lanes" :key="lane.slot" class="flex items-center gap-2">
        <span
          class="w-7 shrink-0 font-mono text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400"
        >
          {{ lane.slot }}
        </span>
        <div
          class="relative h-4 flex-1 overflow-hidden rounded-md bg-slate-100 dark:bg-slate-800/60"
        >
          <div
            v-for="block in lane.blocks"
            :key="block.key"
            class="absolute top-0 bottom-0 rounded-sm"
            :class="block.inFlight ? 'opacity-90' : ''"
            :style="{
              left: `${block.leftPct}%`,
              width: `${block.widthPct}%`,
              backgroundColor: block.color,
              boxShadow: 'inset 0 1px 0 rgba(255,255,255,0.18)',
              borderRight: block.inFlight ? '1px dashed rgba(255,255,255,0.45)' : undefined,
            }"
            :title="block.title"
          />
        </div>
      </div>
    </div>

    <div
      class="mt-1 flex justify-between font-mono text-[10px] text-slate-400 dark:text-slate-500"
      style="padding-left: calc(1.75rem + 0.5rem)"
    >
      <span>&minus;20 s</span>
      <span>now</span>
    </div>

    <div class="mt-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-[11px]">
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
        <div v-for="tenant in props.tenants" :key="tenant.id" class="flex items-center gap-1.5">
          <span
            class="inline-block h-2 w-2 rounded-full"
            :style="{ backgroundColor: tenant.color }"
          />
          <span class="text-slate-700 dark:text-slate-200">{{ tenant.name }}</span>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          :disabled="!props.running"
          :class="[
            'rounded-md px-3 py-1.5 text-xs font-medium text-slate-100',
            'transition-all duration-150 ease-out',
            'enabled:cursor-pointer enabled:active:scale-95',
            'disabled:cursor-not-allowed disabled:opacity-40 disabled:grayscale',
            flashDump
              ? 'bg-emerald-600 ring-2 ring-emerald-300/70 ring-offset-2 ring-offset-white dark:ring-offset-slate-900'
              : 'bg-slate-700 enabled:hover:bg-slate-600',
          ]"
          @click="clickDump"
        >
          <span class="grid">
            <span class="col-start-1 row-start-1" :class="flashDump ? 'invisible' : ''">
              Acme dumps 80
            </span>
            <span class="col-start-1 row-start-1" :class="flashDump ? '' : 'invisible'">
              Dumped ✓
            </span>
          </span>
        </button>
        <button
          type="button"
          :disabled="!props.running"
          :class="[
            'rounded-md px-3 py-1.5 text-xs font-medium text-slate-100',
            'transition-all duration-150 ease-out',
            'enabled:cursor-pointer enabled:active:scale-95',
            'disabled:cursor-not-allowed disabled:opacity-40 disabled:grayscale',
            flashIncident
              ? 'bg-rose-600 ring-2 ring-rose-300/70 ring-offset-2 ring-offset-white dark:ring-offset-slate-900'
              : 'bg-slate-700 enabled:hover:bg-slate-600',
          ]"
          @click="clickIncident"
        >
          <span class="grid">
            <span class="col-start-1 row-start-1" :class="flashIncident ? 'invisible' : ''">
              + P0 incident
            </span>
            <span class="col-start-1 row-start-1" :class="flashIncident ? '' : 'invisible'">
              Incident sent ✓
            </span>
          </span>
        </button>
      </div>
    </div>
  </div>
</template>
