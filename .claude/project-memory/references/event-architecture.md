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

## No workflow-side publishing

Workers never publish events from workflow
scope. The shared `events.NewInterceptor`
registers only an **activity-inbound**
interceptor that publishes
`progress.step.started|completed|failed`
directly to NATS (activity context allows I/O).
Business events stay explicit in activity code
(e.g. `saga.inventory.reserved`).

**Why:** a workflow-scope publish would require
a local activity (the SDK forbids direct I/O
and non-deterministic primitives inside
workflows). That local activity would show up
as a `LocalActivityMarker` in every workflow's
Temporal timeline, cluttering the pedagogical
view. Keeping publishing out of workflow scope
keeps the timeline focused on the pattern's
real activities. No `events.Activity`,
`PublishEvent`, `PublishFromWorkflow`, or
`NewWorkflowEnvelope` exists in the `events`
package — removing any of those is a regression.

## Terminal workflow events are synthesised
## by the Nuxt SSE endpoint

`progress.workflow.completed` and
`progress.workflow.failed` are **not** emitted
by the worker. The SSE endpoint at
`frontend/server/api/patterns/[pattern]/[id]/events.get.ts`
opens a background `handle.result()` watcher on
the Temporal workflow; when the workflow
terminates, the endpoint pushes one synthetic
envelope into the SSE stream (not onto NATS).

Because the SSE stream opens *before* the
workflow is started (so no early events are
missed), the watcher first polls `describe()`
with 250 ms backoff up to 30 s to tolerate
`WorkflowNotFoundError`. Once the workflow
exists, it either emits the terminal event
immediately (if describe already reports a
terminal status) or awaits `result()`. Non-
`COMPLETED` terminal statuses (CANCELLED,
TERMINATED, TIMED_OUT, FAILED) are surfaced as
`progress.workflow.failed` with
`data.error = "workflow <status>"`.

**Why:** this keeps the Temporal timeline
clean while still giving the frontend an
authoritative terminal signal derived from
Temporal itself — no worker plumbing, no race
on startup, no sync needed between the start
endpoint and the SSE endpoint.

## No workflow.started; no compensation bracket

`progress.workflow.started` and
`progress.compensation.started|completed` have
been dropped entirely. The frontend derives
equivalents:

- **Time anchor** — `EventStream.vue` uses
  `events[0].time` as t=0, not a dedicated
  workflow.started envelope.
- **Compensation state** — components set
  `compensating = true` on the first
  `progress.step.failed` whose step is a
  *forward* activity (i.e. belongs to the
  pattern's step list rather than its
  compensation list). Once set, `compensating`
  stays true for the rest of the stream. Under
  that flag, subsequent `progress.step.started`
  events for compensation activities
  (release-inventory, refund-payment,
  cancel-shipment, retract-email) render with
  the amber "warn" tone; the final
  `progress.workflow.failed` synthesised by
  the Nuxt server upgrades the status bar to
  "Saga compensated".

**Why:** in the saga pattern, a workflow-scope
compensation bracket is redundant information —
the compensation activities themselves arrive
as ordinary `progress.step.*` events and their
names are already distinct.

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
