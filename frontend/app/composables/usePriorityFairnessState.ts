import { computed, type ComputedRef, type Ref } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import {
  AGENT_SLOTS,
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
  return AGENT_SLOTS.map((slot) => ({ slot, ticket: null, tenant: null }));
}

function emptyQueues(): Record<TenantId, Ticket[]> {
  return { "mission-critical": [], enterprise: [], business: [] };
}

function freshState(): SimState {
  return {
    queues: emptyQueues(),
    resolved: { "mission-critical": 0, enterprise: 0, business: 0 },
    agents: freshAgents(),
    log: [],
    ticketHistory: [],
    startTime: Date.now(),
  };
}

interface SeededPayload {
  tenants?: Partial<Record<TenantId, Ticket[]>>;
}

/** Payload of helpdesk.ticket.assigned and helpdesk.ticket.resolved. */
interface TicketPayload {
  tenant?: TenantId;
  ticketId?: string;
  priority?: PriorityKey;
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
      // helpdesk.incident.injected is intentionally not consumed here: a P0
      // injection only takes effect in the UI when a worker actually picks
      // the ticket up (via helpdesk.ticket.assigned), exactly like seed
      // tickets. The event remains useful for the event-stream log and
      // code-viewer highlight.
      case "helpdesk.ticket.assigned":
        applyAssigned(state, env.data as TicketPayload, time);
        break;
      case "helpdesk.ticket.resolved":
        applyResolved(state, env.data as TicketPayload, time);
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
    "mission-critical": [...(data.tenants?.["mission-critical"] ?? [])],
    enterprise: [...(data.tenants?.enterprise ?? [])],
    business: [...(data.tenants?.business ?? [])],
  };
}

function applyAssigned(state: SimState, data: TicketPayload, time: number): void {
  if (!data.tenant || !data.agent || !data.ticketId) return;
  const queue = state.queues[data.tenant];
  const idx = queue.findIndex((t) => t.id === data.ticketId);
  const priority = (data.priority ?? 4) as PriorityKey;
  const ticket: Ticket =
    idx >= 0
      ? (queue.splice(idx, 1)[0] as Ticket)
      : { id: data.ticketId, tenant: data.tenant, priority };

  const agent = state.agents.find((a) => a.slot === data.agent);
  if (!agent) return;
  agent.ticket = ticket;
  agent.tenant = data.tenant;
  state.ticketHistory.push({
    ticketId: data.ticketId,
    agent: data.agent,
    tenant: data.tenant,
    priority,
    startTime: time,
    endTime: null,
  });
  // Log entry fires on assignment (the same event that pushes the swim-lane
  // span) so the two visualisations stay synchronised — a row appears in the
  // log at the exact moment its block lands in the chart.
  state.log.unshift({
    time,
    ticket: data.ticketId,
    tenant: data.tenant,
    agent: data.agent,
    priority,
  });
}

function applyResolved(state: SimState, data: TicketPayload, time: number): void {
  if (!data.tenant || !data.agent || !data.ticketId) return;
  const agent = state.agents.find((a) => a.slot === data.agent);
  if (agent) {
    agent.ticket = null;
    agent.tenant = null;
  }
  state.resolved[data.tenant] += 1;

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
