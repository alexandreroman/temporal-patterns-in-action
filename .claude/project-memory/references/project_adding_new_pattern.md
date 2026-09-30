---
name: "Runbook: adding a new pattern"
description: "Six-step checklist for scaffolding a new Temporal pattern across workers/, compose, the workers Makefile, the frontend, and the README."
type: project
---

# Runbook: adding a new pattern

1. `workers/<name>/`: `events.go` (`Pattern` + business type
   constants), `types.go`, `activities.go`, `workflow.go`,
   `workflow_test.go`, and a unique `TaskQueue` constant.
2. `workers/<name>/cmd/worker/main.go` calls `events.RunWorker`,
   registering each activity under its kebab-case name
   ([[event-architecture]]).
3. `workers/<name>/Dockerfile` (copy one, swap the `go build`
   path) and a `worker-<name>` service in `compose.yaml` that
   merges the shared `x-worker` block.
4. Append `<name>` to `PATTERNS` in `workers/Makefile`; the
   `run-%`/`dev-%`/`build-%` rules cover the rest.
5. Frontend: page in `frontend/app/pages/patterns/` (built on
   `usePatternRun` and `PatternHeader`), action routes in
   `frontend/server/api/<name>/`, the `<NAME>_TASK_QUEUE` and
   `<NAME>_WORKFLOW_TYPE` constants in
   `frontend/server/utils/temporal.ts`, request types in
   `frontend/shared/types.ts`, the `<Pattern>*.vue` wrappers
   ([[feedback_frontend_component_conventions]]), and a card plus
   `Icon<Name>.vue` in `frontend/app/pages/index.vue`. The generic
   SSE route already relays the events.
6. Add a row to the Patterns table in `README.md`.

**Why:** each pattern is a self-contained Go package with its own
binary, task queue, and container image, sharing the single
`go.mod` at the `workers/` root; the frontend exposes it in
parallel.

**How to apply:** use this as a checklist when scaffolding a new
pattern. Progress events come for free from the shared
activity-side interceptor — workflow code never publishes. Each
pattern owns an independent Dockerfile; compose services reference
their own `dockerfile:` path rather than sharing one parametrised
by an `ARG PATTERN` build arg.
