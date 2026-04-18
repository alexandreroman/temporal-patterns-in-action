import { Client, Connection } from "@temporalio/client";

let cached: Promise<Client> | null = null;

export function getTemporalClient(): Promise<Client> {
  if (cached !== null) return cached;
  cached = (async () => {
    const address = process.env.TEMPORAL_ADDRESS ?? "localhost:7233";
    const namespace = process.env.TEMPORAL_NAMESPACE ?? "default";
    const connection = await Connection.connect({ address });
    return new Client({ connection, namespace });
  })();
  return cached;
}

export const SAGA_TASK_QUEUE = "patterns-saga";
export const SAGA_WORKFLOW_TYPE = "OrderProcessingWorkflow";
