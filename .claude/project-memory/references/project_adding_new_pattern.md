---
name: "Runbook: adding a new pattern"
description: "Five-step checklist for scaffolding a new Temporal pattern across workers/, workers/Makefile, Nuxt pages, and server routes."
type: project
---

# Runbook: adding a new pattern

1. Create `workers/<name>/` with `types.go`,
   `activities.go`, `workflow.go`,
   `workflow_test.go`, and a unique `TaskQueue`
   constant.
2. Add an entrypoint at
   `workers/<name>/cmd/worker/main.go`
   (package `main`) that dials Temporal,
   registers the workflow and activities on
   the pattern's task queue, and calls
   `worker.Run(worker.InterruptCh())`.
3. Add a `run-<name>` target in
   `workers/Makefile` and extend the
   `PATTERNS` variable so the pattern is
   discoverable.
4. Add a Nuxt page under
   `frontend/app/pages/patterns/` and matching
   server routes under
   `frontend/server/api/<name>/` for
   pattern-specific actions (start, query, …).
   The generic SSE stream at
   `frontend/server/api/patterns/[pattern]/[id]/events.get.ts`
   already relays NATS events for any pattern —
   no new route needed to get a live timeline.
5. Extend the list in
   `frontend/app/pages/index.vue`.

**Why:** Each pattern is a self-contained Go
package with its own binary and task queue;
the frontend is updated in parallel so the UI
exposes the new pattern. All patterns share a
single `go.mod` at the `workers/` root.

**How to apply:** Use this as a checklist when
the user asks to scaffold a new pattern.
Cross-check the event-architecture memory for
kebab-case activity registration. Progress
events come for free from the shared
activity-side interceptor — workflow code must
not publish anything itself. The
`workers/Dockerfile` is already parametrised
via `ARG PATTERN` and needs no change to
support a new pattern.
