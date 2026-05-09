<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from "vue";
import {
  AGENT_SLOTS,
  HISTORY_LEN,
  LOG_CAP,
  NUM_AGENTS,
  PRIORITIES,
  pickRandomTenant,
  randomDuration,
  resetTicketIds,
  seedQueue,
  TENANTS,
  TICK_MS,
  nextTicketId,
  type Agent,
  type LogEntry,
  type PriorityKey,
  type SimState,
  type TenantId,
  type Ticket,
} from "~/utils/priority-fairness";

useSeoMeta({ title: "Priority and Fairness" });

/**
 * Client-side simulation of a multi-tenant helpdesk. Every TICK_MS we
 * advance in-flight tickets, free finished agents, then run a selection
 * algorithm to assign idle agents from the tenant queues. A separate
 * Temporal worker pool will replace this loop in a follow-up commit; the
 * UI semantics (priority + weighted fairness) match Temporal's `Priority`
 * struct so the visualization stays valid.
 */

interface SeedSpec {
  count: number;
  mix: readonly number[];
}

const INITIAL_SEED: Record<TenantId, SeedSpec> = {
  acme: { count: 35, mix: [5, 20, 55, 20] },
  brick: { count: 12, mix: [8, 25, 50, 17] },
  solo: { count: 5, mix: [10, 30, 50, 10] },
};

const ACME_DUMP_MIX = [2, 10, 78, 10] as const;

function freshAgents(): Agent[] {
  return AGENT_SLOTS.slice(0, NUM_AGENTS).map((slot) => ({
    slot,
    ticket: null,
    tenantId: null,
    progress: 0,
    duration: 0,
  }));
}

function freshState(): SimState {
  resetTicketIds();
  const queues: Record<TenantId, Ticket[]> = {
    acme: seedQueue("acme", INITIAL_SEED.acme.count, INITIAL_SEED.acme.mix),
    brick: seedQueue("brick", INITIAL_SEED.brick.count, INITIAL_SEED.brick.mix),
    solo: seedQueue("solo", INITIAL_SEED.solo.count, INITIAL_SEED.solo.mix),
  };
  return {
    queues,
    resolved: { acme: 0, brick: 0, solo: 0 },
    inflight: { acme: 0, brick: 0, solo: 0 },
    agents: freshAgents(),
    log: [],
    history: [],
    startTime: Date.now(),
    fairnessOn: true,
  };
}

const state = reactive<SimState>(freshState());
// Mirror fairnessOn into a top-level ref for ergonomic template binding.
const fairnessOn = ref(state.fairnessOn);

let timer: ReturnType<typeof setInterval> | null = null;

/**
 * Pick the next ticket for an idle agent. Returns the chosen tenantId or
 * null if every queue is empty.
 *
 * Algorithm:
 *  1. Find the lowest priorityKey across non-empty queue heads.
 *  2. Filter to tenants whose head matches that priority.
 *  3. With fairness off (or only one candidate): pick the first in TENANTS order.
 *  4. With fairness on: minimise (resolved + inflight) / weight.
 *     Ties broken by TENANTS order.
 */
function pickTenant(): TenantId | null {
  let bestPriority: PriorityKey | null = null;
  for (const tenant of TENANTS) {
    const head = state.queues[tenant.id]?.[0];
    if (!head) continue;
    if (bestPriority === null || head.priorityKey < bestPriority) {
      bestPriority = head.priorityKey;
    }
  }
  if (bestPriority === null) return null;

  const candidates: TenantId[] = [];
  for (const tenant of TENANTS) {
    const head = state.queues[tenant.id]?.[0];
    if (head && head.priorityKey === bestPriority) candidates.push(tenant.id);
  }
  if (candidates.length === 0) return null;
  if (candidates.length === 1 || !state.fairnessOn) return candidates[0] ?? null;

  let best: TenantId = candidates[0]!;
  let bestScore = scoreFor(best);
  for (let i = 1; i < candidates.length; i++) {
    const id = candidates[i]!;
    const score = scoreFor(id);
    if (score < bestScore) {
      best = id;
      bestScore = score;
    }
  }
  return best;
}

function scoreFor(id: TenantId): number {
  const tenant = TENANTS.find((t) => t.id === id);
  const weight = tenant?.weight ?? 1;
  return (state.resolved[id] + state.inflight[id]) / weight;
}

function assignIdleAgents(): void {
  for (const agent of state.agents) {
    if (agent.ticket) continue;
    const tenantId = pickTenant();
    if (!tenantId) return;
    const queue = state.queues[tenantId];
    const ticket = queue.shift();
    if (!ticket) continue;
    agent.ticket = ticket;
    agent.tenantId = tenantId;
    agent.progress = 0;
    agent.duration = randomDuration();
    state.inflight[tenantId] += 1;
  }
}

function tick(): void {
  const delta: Record<TenantId, number> = { acme: 0, brick: 0, solo: 0 };

  for (const agent of state.agents) {
    if (!agent.ticket || !agent.tenantId) continue;
    agent.progress += 1;
    if (agent.progress >= agent.duration) {
      const tenantId = agent.tenantId;
      const ticketId = agent.ticket.id;
      const priorityKey = agent.ticket.priorityKey;
      state.resolved[tenantId] += 1;
      state.inflight[tenantId] = Math.max(0, state.inflight[tenantId] - 1);
      delta[tenantId] += 1;
      const entry: LogEntry = {
        time: Date.now(),
        ticket: ticketId,
        tenantId,
        agent: agent.slot,
        priorityKey,
      };
      state.log.unshift(entry);
      agent.ticket = null;
      agent.tenantId = null;
      agent.progress = 0;
      agent.duration = 0;
    }
  }

  if (state.log.length > LOG_CAP) state.log.length = LOG_CAP;

  assignIdleAgents();

  state.history.push({ acme: delta.acme, brick: delta.brick, solo: delta.solo });
  if (state.history.length > HISTORY_LEN) state.history.shift();
}

function dumpAcme(): void {
  const tickets = seedQueue("acme", 80, ACME_DUMP_MIX);
  state.queues.acme.push(...tickets);
}

function injectIncident(): void {
  const tenantId = pickRandomTenant();
  state.queues[tenantId].unshift({
    id: nextTicketId(),
    tenantId,
    priorityKey: 1,
  });
}

function toggleFairness(): void {
  state.fairnessOn = !state.fairnessOn;
  fairnessOn.value = state.fairnessOn;
}

function reset(): void {
  const next = freshState();
  state.queues = next.queues;
  state.resolved = next.resolved;
  state.inflight = next.inflight;
  state.agents = next.agents;
  state.log = next.log;
  state.history = next.history;
  state.startTime = next.startTime;
  state.fairnessOn = next.fairnessOn;
  fairnessOn.value = state.fairnessOn;
}

onMounted(() => {
  timer = setInterval(tick, TICK_MS);
});

onBeforeUnmount(() => {
  if (timer !== null) {
    clearInterval(timer);
    timer = null;
  }
});
</script>

<template>
  <section>
    <NuxtLink to="/" class="text-sm text-slate-400 hover:text-slate-100"> &larr; back </NuxtLink>

    <!-- Header row -->
    <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
        >
          <IconPriorityFairness class="h-5 w-5" />
        </span>
        <div class="flex flex-col">
          <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
            Priority and Fairness &mdash; Multi-Tenant Helpdesk
          </h1>
          <p class="text-xs text-slate-400">
            4 agents &middot; 3 tenants &middot; 4 priority levels
          </p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="cursor-pointer rounded-md bg-slate-700 px-3 py-1.5 text-xs font-medium text-slate-100 transition-colors hover:bg-slate-600"
          @click="dumpAcme"
        >
          Acme dumps 80
        </button>
        <button
          type="button"
          class="cursor-pointer rounded-md bg-slate-700 px-3 py-1.5 text-xs font-medium text-slate-100 transition-colors hover:bg-slate-600"
          @click="injectIncident"
        >
          + P0 incident
        </button>
        <button
          type="button"
          class="cursor-pointer rounded-md px-3 py-1.5 text-xs font-medium text-white transition-colors"
          :class="
            fairnessOn ? 'bg-emerald-600 hover:bg-emerald-500' : 'bg-slate-700 hover:bg-slate-600'
          "
          @click="toggleFairness"
        >
          Fairness: {{ fairnessOn ? "ON" : "OFF" }}
        </button>
        <button
          type="button"
          class="cursor-pointer rounded-md bg-slate-700 px-3 py-1.5 text-xs font-medium text-slate-100 transition-colors hover:bg-slate-600"
          @click="reset"
        >
          Reset
        </button>
      </div>
    </div>

    <!-- Throughput chart -->
    <div class="mt-3">
      <PriorityFairnessChart :history="state.history" :tenants="TENANTS" />
    </div>

    <!-- Two-column body -->
    <div class="mt-3 grid gap-3 lg:grid-cols-[2fr_1fr]">
      <div class="flex flex-col gap-3">
        <PriorityFairnessTenants :tenants="TENANTS" :state="state" />
        <PriorityFairnessWorkers :agents="state.agents" />
      </div>
      <div>
        <PriorityFairnessLog :log="state.log" :start-time="state.startTime" />
      </div>
    </div>

    <!-- Legend bar -->
    <div
      class="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2 dark:border-slate-700 dark:bg-slate-900"
    >
      <div class="flex flex-wrap items-center gap-3">
        <span
          v-for="p in PRIORITIES"
          :key="p.key"
          class="inline-flex items-center gap-1.5 text-[11px]"
        >
          <span
            class="rounded-md px-1.5 py-0.5 font-mono text-[10px] tabular-nums"
            :style="{ backgroundColor: p.bg, color: p.fg }"
          >
            {{ p.label }}
          </span>
          <span class="text-slate-500 dark:text-slate-400">{{ p.meaning }}</span>
        </span>
      </div>
      <div class="font-mono text-[11px] text-slate-500 dark:text-slate-400">
        fairnessKey = tenant &middot; weight = contract tier
      </div>
    </div>

    <!-- Notes -->
    <div class="mt-3">
      <PriorityFairnessNotes />
    </div>
  </section>
</template>
