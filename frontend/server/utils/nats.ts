import { connect, type NatsConnection } from "@nats-io/transport-node";
import { isEventEnvelope, type EventEnvelope } from "~~/shared/events";

let cached: Promise<NatsConnection> | null = null;

export function getNatsConnection(): Promise<NatsConnection> {
  if (cached !== null) return cached;
  const config = useRuntimeConfig();
  cached = connect({ servers: config.natsUrl }).catch((error) => {
    cached = null;
    throw error;
  });
  return cached;
}

export async function subscribe(
  subject: string,
  onMessage: (envelope: EventEnvelope) => void,
): Promise<() => void> {
  const nc = await getNatsConnection();
  const sub = nc.subscribe(subject);

  (async () => {
    for await (const msg of sub) {
      let parsed: unknown;
      try {
        parsed = msg.json();
      } catch (error) {
        console.warn("nats: invalid JSON payload", { subject: msg.subject, error });
        continue;
      }
      if (!isEventEnvelope(parsed)) {
        console.warn("nats: dropping malformed envelope", { subject: msg.subject });
        continue;
      }
      onMessage(parsed);
    }
  })().catch((error) => {
    console.warn("nats: subscription iterator failed", { subject, error });
  });

  return () => {
    sub.unsubscribe();
  };
}
