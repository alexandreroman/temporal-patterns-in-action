import { connect, type NatsConnection } from "@nats-io/transport-node";
import { isEventEnvelope, type EventEnvelope } from "~~/shared/events";

let cached: Promise<NatsConnection> | null = null;

export function getNatsConnection(): Promise<NatsConnection> {
  if (cached !== null) return cached;
  const servers = process.env.NATS_URL ?? "nats://localhost:4222";
  cached = connect({ servers }).catch((error) => {
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
  // nc.subscribe() only queues the SUB protocol frame; it returns before the
  // server has registered the interest. Without this flush, a publisher
  // racing the client (e.g. a workflow started immediately after
  // EventSource.onopen) can land its first events on the server before the
  // SUB is processed, and those messages are dropped — symptomatically the
  // priority-fairness `helpdesk.run.seeded` event was missing while later
  // `helpdesk.ticket.assigned` events arrived normally.
  await nc.flush();

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
