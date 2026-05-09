/**
 * Domain model and pure helpers for the Priority and Fairness pattern UI.
 * The simulation is purely client-side; this module owns the constants,
 * narrow types, and small functions that the page and child components share.
 */
import type { EventEnvelope } from "~~/shared/events";

export type TenantId = "acme" | "brick" | "solo";

export interface Tenant {
  id: TenantId;
  name: string;
  tier: string;
  weight: number;
  /** Hex string used for left border, progress bar, chart line, log pill. */
  color: string;
}

export const TENANTS = [
  { id: "acme", name: "Acme Corp", tier: "Mission Critical", weight: 10, color: "#9C5BD9" },
  { id: "brick", name: "BrickLab", tier: "Enterprise", weight: 3, color: "#1A8870" },
  { id: "solo", name: "SoloDev", tier: "Business", weight: 1, color: "#C5803A" },
] as const satisfies readonly Tenant[];

export type PriorityKey = 1 | 2 | 3 | 4;

export interface PriorityLevel {
  key: PriorityKey;
  label: "P0" | "P1" | "P2" | "P3";
  meaning: "incident" | "high" | "normal" | "low";
  bg: string;
  fg: string;
}

export const PRIORITIES = [
  { key: 1, label: "P0", meaning: "incident", bg: "#E8513C", fg: "#FFFFFF" },
  { key: 2, label: "P1", meaning: "high", bg: "#EF9F27", fg: "#2A1502" },
  { key: 3, label: "P2", meaning: "normal", bg: "#6F92E0", fg: "#FFFFFF" },
  { key: 4, label: "P3", meaning: "low", bg: "#B0AEA4", fg: "#2A2A28" },
] as const satisfies readonly PriorityLevel[];

export const NUM_AGENTS = 4;
export const TICK_MS = 250;
export const HISTORY_LEN = 80; // 20 s × 4 ticks/s
export const LOG_CAP = 80;
export const TICKET_HISTORY_CAP = 256;
export const TICKET_DUR_MIN = 4; // ticks
export const TICKET_DUR_MAX = 6;

export type AgentSlot = "a1" | "a2" | "a3" | "a4";
export const AGENT_SLOTS: readonly AgentSlot[] = ["a1", "a2", "a3", "a4"];

export interface Ticket {
  id: string; // 4-digit zero-padded
  tenantId: TenantId;
  priorityKey: PriorityKey;
}

export interface Agent {
  slot: AgentSlot;
  ticket: Ticket | null;
  tenantId: TenantId | null;
  progress: number;
  duration: number;
}

export interface LogEntry {
  /** Milliseconds since the simulation's startTime — formatted as MM:SS.d. */
  time: number;
  ticket: string;
  tenantId: TenantId;
  agent: AgentSlot;
  priorityKey: PriorityKey;
}

export interface HistorySample {
  acme: number;
  brick: number;
  solo: number;
}

export interface TicketSpan {
  ticketId: string;
  agent: AgentSlot;
  tenantId: TenantId;
  priorityKey: PriorityKey;
  /** ms epoch — assignment time. */
  startTime: number;
  /** ms epoch — resolution time, or null while still in-flight. */
  endTime: number | null;
}

export interface SimState {
  queues: Record<TenantId, Ticket[]>;
  resolved: Record<TenantId, number>;
  inflight: Record<TenantId, number>;
  agents: Agent[];
  log: LogEntry[];
  history: HistorySample[];
  ticketHistory: TicketSpan[];
  startTime: number;
  fairnessOn: boolean;
}

const PRIORITY_KEYS: readonly PriorityKey[] = [1, 2, 3, 4];

// Module-level counter so ticket IDs stay unique across queue refills.
// Resets when `resetTicketIds()` is called from the page on Reset.
let ticketCounter = 0;

export function resetTicketIds(): void {
  ticketCounter = 0;
}

export function nextTicketId(): string {
  ticketCounter = (ticketCounter + 1) % 10_000;
  return ticketCounter.toString().padStart(4, "0");
}

/**
 * Pick a priority key from a 4-bucket percentage distribution. The mix is
 * not required to sum to exactly 100 — we sample within its actual total.
 */
export function pickFromMix(mix: readonly number[]): PriorityKey {
  const total = mix.reduce((s, n) => s + n, 0);
  let r = Math.random() * total;
  for (let i = 0; i < PRIORITY_KEYS.length; i++) {
    r -= mix[i] ?? 0;
    if (r <= 0) return PRIORITY_KEYS[i] ?? 4;
  }
  return PRIORITY_KEYS[PRIORITY_KEYS.length - 1] ?? 4;
}

export function randomDuration(): number {
  const span = TICKET_DUR_MAX - TICKET_DUR_MIN + 1;
  return TICKET_DUR_MIN + Math.floor(Math.random() * span);
}

export function pickRandomTenant(): TenantId {
  const idx = Math.floor(Math.random() * TENANTS.length);
  return (TENANTS[idx]?.id ?? "acme") as TenantId;
}

/**
 * Build `count` tickets for a tenant using the given priority mix. The
 * resulting array is ordered as it should be appended to the tenant's queue:
 * higher-priority tickets are NOT pre-sorted, since the dispatcher already
 * picks by priority across queue heads. We do, however, sort within the
 * batch so that priority bursts (e.g. the +80 dump) feel realistic — newest
 * first by priority, then arrival.
 */
export function seedQueue(tenantId: TenantId, count: number, mix: readonly number[]): Ticket[] {
  const tickets: Ticket[] = [];
  for (let i = 0; i < count; i++) {
    tickets.push({
      id: nextTicketId(),
      tenantId,
      priorityKey: pickFromMix(mix),
    });
  }
  // Sort the seeded batch by priority so the lane visually shows the most
  // urgent items at the head — the dispatcher's algorithm is independent.
  tickets.sort((a, b) => a.priorityKey - b.priorityKey);
  return tickets;
}

export function priorityLevel(key: PriorityKey): PriorityLevel {
  return PRIORITIES[key - 1] ?? PRIORITIES[3]!;
}

export function tenantById(id: TenantId): Tenant {
  const found = TENANTS.find((t) => t.id === id);
  if (!found) throw new Error(`unknown tenant ${id}`);
  return found;
}

// Module-level counter so envelope IDs stay unique across runs in this tab.
let envelopeCounter = 0;

/**
 * Synthesize an EventEnvelope locally — the simulation has no backend, but
 * we shape its outputs the same way the workers do so the EventStream and
 * CodeViewer wrappers can reuse the generic pattern shells.
 */
export function buildEvent<T>(workflowId: string, type: string, data: T): EventEnvelope<T> {
  envelopeCounter += 1;
  return {
    specversion: "1.0",
    id: `pf-${envelopeCounter}`,
    source: "ui-sim",
    type,
    workflowId,
    runId: workflowId,
    time: new Date().toISOString(),
    data,
  };
}

/**
 * Format a millisecond duration as MM:SS.d (deciseconds), used by the log.
 */
export function formatLogTime(ms: number): string {
  const totalDeci = Math.max(0, Math.floor(ms / 100));
  const deci = totalDeci % 10;
  const totalSec = Math.floor(totalDeci / 10);
  const sec = totalSec % 60;
  const min = Math.floor(totalSec / 60);
  return `${min.toString().padStart(2, "0")}:${sec.toString().padStart(2, "0")}.${deci}`;
}
