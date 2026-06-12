---
name: "SSE endpoints need an immediate initial push"
description: "Every pattern SSE endpoint must push one message right after subscribing so Node/h3 flushes the response headers and the client's EventSource.onopen fires in ms, not seconds."
type: feedback
---

# SSE endpoints need an immediate initial push

When a pattern SSE endpoint is written with
`createEventStream(event)` in Nitro/h3, **push
one message (heartbeat or `ready` sentinel)
immediately after the NATS subscription is set
up**, before returning `stream.send()`. Without
it, the response headers are NOT flushed until
the first chunk is written, and
`EventSource.onopen` on the client waits for
that first chunk.

**Why:** Node/h3 buffers `writeHead` until a
body write happens. Without an open-time push,
headers stay held until the 15 s heartbeat
interval fires. Because the client's
`waitForStreamOpen()` in
`frontend/app/pages/patterns/saga.vue` waits
for `onopen` before POSTing `/api/saga/start`,
the whole start is delayed by ~15 s ("workflows
take several seconds to start"). With the
immediate push it drops to ~8 ms on the
preview (prod) build.

**How to apply:** any new pattern adding its
own SSE endpoint MUST push a first chunk
immediately after `subscribe(...)` returns,
not rely on the heartbeat interval. Keep
`HEARTBEAT_INTERVAL_MS` (15 s) for liveness
only. The existing endpoint at
`frontend/server/api/patterns/[pattern]/[id]/events.get.ts`
is the reference implementation — call
`pushHeartbeat()` once, then start the
interval.
