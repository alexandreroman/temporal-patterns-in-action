export type NodeState = "idle" | "active" | "ok" | "warn" | "error";
export type EdgeState = "idle" | "active" | "warn" | "error";

export type NodeKey = "ui" | "temporal" | "worker" | "s1" | "s2" | "s3" | "s4";
export type EdgeKey = "ui_tmp" | "tmp_wk" | "wk_s1" | "wk_s2" | "wk_s3" | "wk_s4";

export type Nodes = Record<NodeKey, NodeState>;
export type Edges = Record<EdgeKey, EdgeState>;

export interface ArchState {
  nodes: Nodes;
  edges: Edges;
  running: boolean;
}
