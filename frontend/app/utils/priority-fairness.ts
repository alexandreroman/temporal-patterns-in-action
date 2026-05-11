/**
 * Domain model and pure helpers for the Priority and Fairness pattern UI.
 * The page consumes live SSE events from the Temporal worker; this module
 * owns the constants, narrow types, and small functions that the page and
 * its child components share.
 */

export type TenantId = "acme" | "brick" | "solo";

export interface Tenant {
  id: TenantId;
  /** Contract tier — displayed as the tenant's name in the UI. */
  name: string;
  weight: number;
  /** Hex string used for left border, progress bar, chart line, log pill. */
  color: string;
}

export const TENANTS = [
  { id: "acme", name: "Mission Critical", weight: 10, color: "#9C5BD9" },
  { id: "brick", name: "Enterprise", weight: 3, color: "#1A8870" },
  { id: "solo", name: "Business", weight: 1, color: "#C5803A" },
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
  /** Milliseconds since the run's startTime — formatted as MM:SS.d. */
  time: number;
  ticket: string;
  tenantId: TenantId;
  agent: AgentSlot;
  priorityKey: PriorityKey;
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
  agents: Agent[];
  log: LogEntry[];
  ticketHistory: TicketSpan[];
  startTime: number;
}

export function priorityLevel(key: PriorityKey): PriorityLevel {
  return PRIORITIES[key - 1] ?? PRIORITIES[3]!;
}

export function tenantById(id: TenantId): Tenant {
  const found = TENANTS.find((t) => t.id === id);
  if (!found) throw new Error(`unknown tenant ${id}`);
  return found;
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
