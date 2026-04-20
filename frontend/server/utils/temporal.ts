import { Client, Connection } from "@temporalio/client";
import { DEMO_KEY, EncryptionCodec } from "./encryption-codec";

let plainClient: Promise<Client> | null = null;
let encryptedClient: Promise<Client> | null = null;

async function buildConnection(): Promise<Connection> {
  const address = process.env.TEMPORAL_ADDRESS ?? "localhost:7233";
  return Connection.connect({ address });
}

function namespace(): string {
  return process.env.TEMPORAL_NAMESPACE ?? "default";
}

export function getTemporalClient(): Promise<Client> {
  if (plainClient !== null) return plainClient;
  plainClient = (async () => {
    const connection = await buildConnection();
    return new Client({ connection, namespace: namespace() });
  })();
  return plainClient;
}

// Returns a Client whose data converter applies the AES-256-GCM codec on the
// way to Temporal. Used only by the encryption pattern's encrypted scenario;
// every other caller uses getTemporalClient() and sees raw payloads.
export function getEncryptedTemporalClient(): Promise<Client> {
  if (encryptedClient !== null) return encryptedClient;
  encryptedClient = (async () => {
    const connection = await buildConnection();
    return new Client({
      connection,
      namespace: namespace(),
      dataConverter: { payloadCodecs: [new EncryptionCodec(DEMO_KEY)] },
    });
  })();
  return encryptedClient;
}

export const SAGA_TASK_QUEUE = "patterns-saga";
export const SAGA_WORKFLOW_TYPE = "OrderProcessingWorkflow";

export const BATCH_TASK_QUEUE = "patterns-batch";
export const BATCH_WORKFLOW_TYPE = "BatchProcessingWorkflow";

export const ENCRYPTION_TASK_QUEUE_CLEAR = "patterns-encryption-clear";
export const ENCRYPTION_TASK_QUEUE_ENCRYPTED = "patterns-encryption-encrypted";
export const ENCRYPTION_WORKFLOW_TYPE = "ProcessSensitiveOrderWorkflow";

export const AGENT_TASK_QUEUE = "patterns-agent";
export const AGENT_WORKFLOW_TYPE = "TravelAgentWorkflow";
export const AGENT_APPROVAL_SIGNAL = "approval";
