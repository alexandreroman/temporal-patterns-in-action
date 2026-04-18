---
name: Event architecture (NATS)
description: Cross-module contract and design decisions for workflow progress and business events published over NATS from Go workers to the Nuxt frontend.
type: project
---

# Event architecture (NATS)

The project uses **core NATS** (not JetStream) as
an event bus between Go workers and the Nuxt
frontend. The subject hierarchy is already
JetStream-ready if durability or replay is
needed later.

## Cross-module contract

**Subject:**
`patterns.<pattern>.<workflowId>.<category>`
where `<category>` is `progress` or `business`.
Wildcards: `patterns.<pattern>.<id>.>` for a
per-workflow stream, `patterns.<pattern>.*.progress`
for a pattern-wide progression feed,
`patterns.<pattern>.*.business` for a
pattern-wide business feed, `patterns.>` for
cluster-wide observation.

**Envelope** (CloudEvents-inspired JSON):
`{ specversion, id (UUIDv4), source
("patterns.<pattern>"), type, workflowId, runId,
time (RFC3339 UTC ms), data }`. Category is
derived from the type — never stored as a
separate field — via the rule:
`HasPrefix(type, "progress.")` → `progress`,
otherwise → `business`. Progress types follow
`progress.<subtype>` (shared across patterns);
business types follow `<pattern>.<subtype>`
(e.g. `saga.inventory.reserved`). The
asymmetry is intentional: pattern-prefixing
business types eliminates any possible
`type`-string collision between patterns, while
progress types stay pattern-agnostic because
they are emitted by a shared interceptor.

**Why these shapes:** the subject encodes
routing metadata for NATS-side filtering
without payload parsing, and the envelope is
self-describing so the frontend can render
events without hard-coded coupling to a pattern.

## Progress vs business split

`progress.*` events (`workflow.started`,
`step.started|completed|failed`,
`compensation.started|completed`,
`workflow.completed|failed`) are emitted
automatically by a shared Temporal
`WorkerInterceptor`. Every new pattern gets
timeline tracking for free.

Business events (e.g.
`saga.inventory.reserved`,
`saga.payment.charged`) stay explicit in
activity code — they are the pedagogical
payload of each pattern and the interceptor
cannot infer them. Each pattern uses its own
name as the type prefix so business types
never collide across patterns.

**Why:** ~90 % of the lifecycle boilerplate
disappears from business code, while the
pattern-specific semantics remain visible where
they are authored. Adding a new pattern
requires zero extra wiring to get a timeline.

## Determinism rule for workflow-side publishing

The **workflow-inbound** interceptor publishes
via
`workflow.ExecuteLocalActivity(ctx,
"PublishEvent", env)`.

The **activity-inbound** interceptor publishes
directly to NATS (activity context allows I/O).

**Why:** workflow code must stay deterministic,
so no direct NATS calls or `time.Now()` /
`uuid.NewString()` from workflow scope. Fresh
UUID + RFC3339 timestamp are assigned inside
the `PublishEvent` activity. Every worker
registers `&events.Activity{Publisher}` to
expose `PublishEvent`.

## Step-naming convention (gotcha)

Activities MUST be registered with explicit
kebab-case names via
`RegisterActivityWithOptions(method,
activity.RegisterOptions{Name: "..."})` — not
via the struct form `RegisterActivity(&Activities{})`,
which would expose Go method names.

**Why:** the `step` field in `progress.step.*`
events comes from
`activity.GetInfo().ActivityType.Name`; the
frontend timeline and the pre-existing saga
`Progress.CurrentStep` query use kebab-case.
Using the struct form would publish `ReserveCar`
while the UI expects `reserve-car`, silently
breaking the timeline.

**How to apply:** when adding a new pattern,
register each activity method individually with
its canonical kebab-case name. The Go SDK
resolves function-value calls
(`workflow.ExecuteActivity(ctx, a.Method, ...)`)
to the custom registered name automatically, so
workflows do not need string-based activity
references.

## Publisher fallback

`events.NewPublisher(url)` returns a
`NopPublisher` when the URL is empty or the
NATS dial fails. Workers stay runnable without
NATS for local dev and unit tests.

**Why:** demos must not require the full infra
to come up. Tests pass with a nil or Nop
publisher. A failing NATS does not kill the
worker.

## When to migrate to JetStream

Consider JetStream when:

- A new pattern requires event replay for late
  subscribers.
- Multiple consumers must each see the full
  stream (durable consumers).
- Events must survive a NATS restart.

Migration plan: declare a stream per pattern
(`stream=patterns-<name>,
subjects=patterns.<name>.>`) and switch the
publisher to `js.Publish`. No subject or
envelope changes required.
