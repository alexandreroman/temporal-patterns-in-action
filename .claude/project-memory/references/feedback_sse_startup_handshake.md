---
name: "SSE start-up handshake"
description: "A run starts only after its SSE subscription is live: subscribe, flush, first push on the server; a flush:'sync' watch on the client."
type: feedback
---

# SSE start-up handshake

A pattern page opens its SSE stream, awaits `waitForOpen()` from
`frontend/app/composables/usePatternStream.ts`, then POSTs
`/api/<pattern>/start` (sequenced by `usePatternRun`). NATS silently
drops events published before the SUB is live, so three links must
hold, in order:

1. `subscribe()` in `frontend/server/utils/nats.ts` awaits
   `nc.flush()` after `nc.subscribe()`, which only queues the SUB.
2. `events.get.ts` pushes one heartbeat right after subscribing;
   Node/h3 holds headers until the first chunk, so `onopen` would
   otherwise wait for the 15 s interval.
3. The `(pattern, workflowId)` watch in `usePatternStream.ts` uses
   `{ immediate: true, flush: "sync" }`, and the caller sets
   `workflowId` in the same tick as `waitForOpen()`. With the
   default `pre` flush, `waitForOpen()` sees the previous run's
   `open` status and the POST races the new subscription.

**Why:** a missing link shows up as a lost first event (e.g.
`helpdesk.run.seeded`: priority-fairness tenant queues stay empty
while the swimlane fills) or a ~15 s start delay. No static check
catches either.

**How to apply:** any SSE or pub/sub pipeline whose caller starts
the publisher right after opening the subscriber keeps all three
links.
