import { createRequire } from "node:module";
import { resolve } from "node:path";
import { DEMO_KEY, EncryptionCodec } from "./encryption-codec";
import type * as TemporalClient from "@temporalio/client";

// @temporalio/client goes through Node's CommonJS loader instead of a plain
// `import`: since 1.16 the package makes Nitro emit extension-less ESM imports
// (".../lib/async-completion-client") that Node's loader rejects, so every SSR
// route answers HTTP 500 in dev. createRequire loads the package as the
// CommonJS it actually is, bypassing the bundler's ESM transform.
//
// Two details keep the production build working too:
//   - resolution is anchored on the running server entry, which sits next to
//     the node_modules tree holding the package. Nitro rewrites
//     `import.meta.url` to the placeholder "file:///_entry.js" in non-entry
//     chunks, so using it here would search from the filesystem root;
//   - `nitro.externals.traceInclude` in nuxt.config.ts ships the package in
//     .output/server/node_modules, since a require is invisible to Nitro's
//     dependency tracer.

// Node always sets argv[1]; the fallback only satisfies the type checker.
const serverEntry = resolve(process.argv[1] ?? "index.mjs");
const requireFromServerEntry = createRequire(serverEntry);

const { Client, Connection, WorkflowNotFoundError } = requireFromServerEntry(
  "@temporalio/client",
) as typeof TemporalClient;

// The binding above is a value only; alias the class so the same name keeps
// working as a type.
type Client = TemporalClient.Client;

// Re-exported so the whole server shares this single load: a second require
// would yield a distinct class identity and silently break `instanceof`.
export { WorkflowNotFoundError };

let plainClient: Promise<Client> | null = null;

export function getTemporalClient(): Promise<Client> {
  if (plainClient !== null) return plainClient;
  const address = process.env.TEMPORAL_ADDRESS ?? "localhost:7233";
  const namespace = process.env.TEMPORAL_NAMESPACE ?? "default";
  plainClient = Connection.connect({ address })
    .then((connection) => new Client({ connection, namespace }))
    .catch((error) => {
      // Drop the failed dial so the next caller retries instead of reusing
      // the rejected promise forever.
      plainClient = null;
      throw error;
    });
  return plainClient;
}

// Returns a Client whose data converter applies the AES-256-GCM codec on the
// way to Temporal. Used only by the encryption pattern's encrypted scenario;
// every other caller uses getTemporalClient() and sees raw payloads.
//
// Clients are cheap, connections are not: this one shares the plain client's
// connection and namespace instead of dialing Temporal a second time.
export async function getEncryptedTemporalClient(): Promise<Client> {
  const plain = await getTemporalClient();
  return new Client({
    connection: plain.connection,
    namespace: plain.options.namespace,
    dataConverter: { payloadCodecs: [new EncryptionCodec(DEMO_KEY)] },
  });
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

export const MULTI_AGENT_TASK_QUEUE = "patterns-multi-agent";
export const MULTI_AGENT_WORKFLOW_TYPE = "DeepResearchWorkflow";

export const ENTITY_TASK_QUEUE = "patterns-entity";
export const ENTITY_WORKFLOW_TYPE = "ShoppingCartWorkflow";
export const ENTITY_SIGNAL_ADD_ITEM = "addItem";
export const ENTITY_SIGNAL_UPDATE_QTY = "updateQty";
export const ENTITY_SIGNAL_REMOVE_ITEM = "removeItem";
export const ENTITY_SIGNAL_CHECKOUT = "checkout";
export const ENTITY_QUERY_GET_CART = "getCart";

export const PRIORITY_FAIRNESS_TASK_QUEUE = "patterns-priority-fairness";
export const PRIORITY_FAIRNESS_WORKFLOW_TYPE = "HelpdeskRunWorkflow";
export const PRIORITY_FAIRNESS_SIGNAL_INCIDENT = "inject-p0-incident";
