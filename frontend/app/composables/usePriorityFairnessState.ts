import { computed, type ComputedRef, type Ref } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import {
  AGENT_SLOTS,
  LOG_CAP,
  NUM_AGENTS,
  TICKET_HISTORY_CAP,
  type Agent,
  type AgentSlot,
  type PriorityKey,
  type SimState,
  type TenantId,
  type Ticket,
} from "~/utils/priority-fairness";

/**
 * Folds the live SSE event stream into the SimState shape the
 * tenants/workers/log/chart components consume. Pure derivation: identical
 * input events yield identical state.
 */
export function usePriorityFairnessState(events: Ref<EventEnvelope[]>): ComputedRef<SimState> {
  return computed(() => deriveState(events.value));
}

function freshAgents(): Agent[] {
  return AGENT_SLOTS.slice(0, NUM_AGENTS).map((slot) => ({
    slot,
    ticket: null,
    tenantId: null,
    progress: 0,
    duration: 0,
  }));
}

function emptyQueues(): Record<TenantId, Ticket[]> {
  return { acme: [], brick: [], solo: [] };
}

function freshState(): SimState {
  return {
    queues: emptyQueues(),
    resolved: { acme: 0, brick: 0, solo: 0 },
    agents: freshAgents(),
    log: [],
    ticketHistory: [],
    startTime: Date.now(),
  };
}

interface SeededPayload {
  tenants?: Partial<Record<TenantId, Ticket[]>>;
}

interface DumpPayload {
  tenantId?: TenantId;
  tickets?: Ticket[];
}

interface IncidentPayload {
  tenantId?: TenantId;
  ticket?: Ticket;
}

interface AssignedPayload {
  tenantId?: TenantId;
  ticketId?: string;
  priorityKey?: PriorityKey;
  agent?: AgentSlot;
}

interface ResolvedPayload {
  tenantId?: TenantId;
  ticketId?: string;
  priorityKey?: PriorityKey;
  agent?: AgentSlot;
}

function deriveState(events: readonly EventEnvelope[]): SimState {
  const state = freshState();

  for (const env of events) {
    const time = new Date(env.time).getTime();
    switch (env.type) {
      case "helpdesk.run.seeded":
        applySeed(state, env.data as SeededPayload, time);
        break;
      case "helpdesk.dump.executed":
        applyDump(state, env.data as DumpPayload);
        break;
      case "helpdesk.incident.injected":
        applyIncident(state, env.data as IncidentPayload);
        break;
      case "helpdesk.ticket.assigned":
        applyAssigned(state, env.data as AssignedPayload, time);
        break;
      case "helpdesk.ticket.resolved":
        applyResolved(state, env.data as ResolvedPayload, time);
        break;
    }
  }

  if (state.ticketHistory.length > TICKET_HISTORY_CAP) {
    state.ticketHistory.splice(0, state.ticketHistory.length - TICKET_HISTORY_CAP);
  }
  return state;
}

function applySeed(state: SimState, data: SeededPayload, time: number): void {
  state.startTime = Number.isFinite(time) ? time : Date.now();
  state.queues = {
    acme: [...(data.tenants?.acme ?? [])],
    brick: [...(data.tenants?.brick ?? [])],
    solo: [...(data.tenants?.solo ?? [])],
  };
}

function applyDump(state: SimState, data: DumpPayload): void {
  if (!data.tenantId || !data.tickets) return;
  state.queues[data.tenantId].push(...data.tickets);
}

function applyIncident(state: SimState, data: IncidentPayload): void {
  if (!data.tenantId || !data.ticket) return;
  state.queues[data.tenantId].unshift(data.ticket);
}

function applyAssigned(state: SimState, data: AssignedPayload, time: number): void {
  if (!data.tenantId || !data.agent || !data.ticketId) return;
  const queue = state.queues[data.tenantId];
  const idx = queue.findIndex((t) => t.id === data.ticketId);
  const ticket: Ticket =
    idx >= 0
      ? (queue.splice(idx, 1)[0] as Ticket)
      : {
          id: data.ticketId,
          tenantId: data.tenantId,
          priorityKey: (data.priorityKey ?? 4) as PriorityKey,
        };

  const agent = state.agents.find((a) => a.slot === data.agent);
  if (!agent) return;
  agent.ticket = ticket;
  agent.tenantId = data.tenantId;
  // duration=1, progress=1 keeps the worker card's bar fully filled while
  // busy — we don't have sub-second progress from the backend, so the bar
  // simply represents "agent occupied" rather than ticket completion %.
  agent.duration = 1;
  agent.progress = 1;
  state.ticketHistory.push({
    ticketId: data.ticketId,
    agent: data.agent,
    tenantId: data.tenantId,
    priorityKey: (data.priorityKey ?? 4) as PriorityKey,
    startTime: time,
    endTime: null,
  });
}

function applyResolved(state: SimState, data: ResolvedPayload, time: number): void {
  if (!data.tenantId || !data.agent || !data.ticketId) return;
  const agent = state.agents.find((a) => a.slot === data.agent);
  if (agent) {
    agent.ticket = null;
    agent.tenantId = null;
    agent.progress = 0;
    agent.duration = 0;
  }
  state.resolved[data.tenantId] += 1;
  state.log.unshift({
    time,
    ticket: data.ticketId,
    tenantId: data.tenantId,
    agent: data.agent,
    priorityKey: (data.priorityKey ?? 4) as PriorityKey,
  });
  if (state.log.length > LOG_CAP) state.log.length = LOG_CAP;

  for (let i = state.ticketHistory.length - 1; i >= 0; i--) {
    const span = state.ticketHistory[i];
    if (
      span &&
      span.endTime === null &&
      span.ticketId === data.ticketId &&
      span.agent === data.agent
    ) {
      span.endTime = time;
      break;
    }
  }
}
