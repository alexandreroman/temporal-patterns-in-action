---
name: "waitForOpen sees stale 'open' across runs"
description: "The pattern-stream watch in usePatternStream must use flush:'sync' so callers that set workflowId and immediately await waitForOpen() don't observe the previous run's open state and POST /start before the new SSE subscription is live."
type: feedback
---

# waitForOpen sees stale `open` status across runs

In `frontend/app/composables/usePatternStream.ts`, the `watch(...)`
that recreates the `EventSource` on every `(pattern, workflowId)`
change MUST run with `flush: "sync"`. The page's `start()` function
sets `workflowId.value = newId` and on the very next line calls
`await waitForOpen()`. With the default `flush: "pre"` scheduler the
watch is queued as a microtask and has not yet run when
`waitForOpen()` samples `status.value` — so on every run after the
first, status is still `"open"` from the previous run's
`EventSource`, the early-resolve path

```ts
if (status.value === "open") return resolve();
```

fires immediately, and `start()` POSTs `/api/priority-fairness/start`
before the new SSE subscription has been established server-side.
The worker then runs `AnnounceRunSeeded` and publishes
`helpdesk.run.seeded` to NATS before the new subscriber's SUB has
been processed, and the seeded event is dropped — symptomatically
the three tenant queue panels stay `(empty)` for the entire run
while the swimlane, agent cards and resolved counts populate
normally from later events.

**Why:** the [[feedback_nats_subscribe_flush]] fix closed the server-side
race (SUB → flush before signalling SSE open), but the client side
still observed a stale `status` from the previous run because the
reactive watch was async. Verified in Chrome (2026-05-12): on the
failing run, `FETCH_START_REQ` fired ~17 ms before the new
`EventSource`'s `open` event; the new SUB therefore wasn't live yet
when the worker published. Setting `flush: "sync"` on the watch
makes the close/open + `status.value = "connecting"` mutation
happen inside the `.value = id` setter, so the very next line in
`start()` correctly observes the new connection state and
`waitForOpen()` waits for the real open transition.

**How to apply:** any composable that exposes a `waitForOpen()`-style
helper alongside a `workflowId`-driven reactive watch must keep the
watch on `flush: "sync"`, or refactor `waitForOpen()` to ignore
status sampled before the watch has had a chance to run (e.g. by
yielding via `await nextTick()` first, or by tying the promise to
the EventSource instance rather than to `status`). When adding a new
pattern that opens its own SSE pipeline, mirror the
`{ immediate: true, flush: "sync" }` option set verbatim.
