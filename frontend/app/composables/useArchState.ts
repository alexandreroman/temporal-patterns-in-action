import type { Edges, EdgeKey, Nodes, NodeKey } from "~/types/architecture";

/**
 * Shared scaffolding for architecture-diagram wrappers.
 * Every pattern uses the same node/edge layout
 * (ui, temporal, worker, s1..s4 + the wiring between them); only the
 * event-to-state switch is pattern-specific and stays in the wrapper.
 */

export const NODE_IDS: NodeKey[] = ["ui", "temporal", "worker", "s1", "s2", "s3", "s4"];
export const EDGE_IDS: EdgeKey[] = ["ui_tmp", "tmp_wk", "wk_s1", "wk_s2", "wk_s3", "wk_s4"];
export const SERVICE_NODES: NodeKey[] = ["s1", "s2", "s3", "s4"];
export const SERVICE_EDGES: EdgeKey[] = ["wk_s1", "wk_s2", "wk_s3", "wk_s4"];

export function initialNodes(): Nodes {
  return {
    ui: "idle",
    temporal: "idle",
    worker: "idle",
    s1: "idle",
    s2: "idle",
    s3: "idle",
    s4: "idle",
  };
}

export function initialEdges(): Edges {
  return {
    ui_tmp: "idle",
    tmp_wk: "idle",
    wk_s1: "idle",
    wk_s2: "idle",
    wk_s3: "idle",
    wk_s4: "idle",
  };
}

export function resetAll(nodes: Nodes, edges: Edges) {
  for (const id of NODE_IDS) nodes[id] = "idle";
  for (const id of EDGE_IDS) edges[id] = "idle";
}

export function resetServices(nodes: Nodes, edges: Edges) {
  for (const id of SERVICE_NODES) {
    if (nodes[id] !== "ok" && nodes[id] !== "error") nodes[id] = "idle";
  }
  for (const id of SERVICE_EDGES) {
    if (edges[id] !== "error") edges[id] = "idle";
  }
}

/**
 * Light the UI→Temporal→Worker strip while a run is in flight. No
 * workflow.started event arrives, so wrappers call this after folding the
 * event stream to anchor the baseline — only nodes/edges still `idle` are
 * promoted to `active`, preserving any error/ok/warn states set by events.
 */
export function applyRunningBaseline(nodes: Nodes, edges: Edges) {
  if (nodes.ui === "idle") nodes.ui = "active";
  if (nodes.temporal === "idle") nodes.temporal = "active";
  if (nodes.worker === "idle") nodes.worker = "active";
  if (edges.ui_tmp === "idle") edges.ui_tmp = "active";
  if (edges.tmp_wk === "idle") edges.tmp_wk = "active";
}

/**
 * Terminal failure: reset per-service slots and paint the UI→Temporal→Worker
 * strip red. Shared by every architecture wrapper's
 * `progress.workflow.failed` branch. Callers still flip their local
 * `running` flag to false.
 */
export function applyWorkflowFailed(nodes: Nodes, edges: Edges) {
  resetServices(nodes, edges);
  nodes.ui = "error";
  nodes.temporal = "error";
  nodes.worker = "error";
  edges.ui_tmp = "error";
  edges.tmp_wk = "error";
}
