// Frozen contract shared between Go workers and the Nuxt frontend.
// Subject format: patterns.<pattern>.<workflowId>.<category>

export interface EventEnvelope<T = unknown> {
  specversion: "1.0";
  id: string;
  source: string;
  type: string;
  workflowId: string;
  runId: string;
  time: string;
  data: T;
}

export function isEventEnvelope(value: unknown): value is EventEnvelope {
  if (value === null || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  return (
    v.specversion === "1.0" &&
    typeof v.id === "string" &&
    typeof v.source === "string" &&
    typeof v.type === "string" &&
    typeof v.workflowId === "string" &&
    typeof v.runId === "string" &&
    typeof v.time === "string" &&
    typeof v.data === "object" &&
    v.data !== null
  );
}
