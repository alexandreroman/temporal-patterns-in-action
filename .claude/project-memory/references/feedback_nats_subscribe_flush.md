---
name: "NATS subscribe must be flushed before signalling SSE open"
description: "Subscribers must await nc.flush() right after nc.subscribe() so the SUB frame is acked by the server before the client races ahead and the publisher starts emitting."
type: feedback
---

# NATS subscribe must be flushed before signalling SSE open

In the SSE endpoint at
`frontend/server/api/patterns/[pattern]/[id]/events.get.ts`,
and in the shared `subscribe()` helper at
`frontend/server/utils/nats.ts`, the
`nc.subscribe(subject)` call only queues the
SUB protocol frame in the client's outbound
buffer — it returns *before* the NATS server
has acknowledged the subscription. The helper
must `await nc.flush()` immediately after
`nc.subscribe(...)` so subscription is live
before any later code returns control to the
client.

**Why:** the bug presented as "with Fairness ON
the Priority pattern sometimes shows no tickets
in the tenant queues, only in the swimlane".
Sequence:

1. The page opens the SSE, awaits `onopen`.
2. The handler subscribes + pushes the initial
   heartbeat (see
   [[feedback_sse_initial_flush]]) — onopen
   fires on the client.
3. The page POSTs `/api/priority-fairness/start`.
4. The worker's `HelpdeskRunWorkflow` runs the
   `AnnounceRunSeeded` local activity and
   publishes `helpdesk.run.seeded` to NATS.

If the SUB from step 2 hasn't reached the NATS
server by the time the worker publishes in
step 4, the server has no matching interest
and silently drops the message. Later
`helpdesk.ticket.assigned` events fire after
the SUB has caught up, so they land normally
— which is exactly why the swimlane fills
while the tenant queues stay empty. Adding
`await nc.flush()` between `nc.subscribe(...)`
and the heartbeat push closes the race: flush
returns only once the server has acked, so the
client's onopen now strictly happens-after the
subscription becomes live.

**How to apply:** every server-side NATS
subscribe in this repo must be followed by
`await nc.flush()` before any "stream is
ready" signal reaches the client. Combine
with [[feedback_sse_initial_flush]] (push an
initial chunk so headers are flushed) — both
are required, in this order: subscribe →
flush → initial push → return stream. If a
future pattern adds another pub/sub
integration where the subscriber and the
publisher are kicked off back-to-back from
the same caller, repeat this pattern.
