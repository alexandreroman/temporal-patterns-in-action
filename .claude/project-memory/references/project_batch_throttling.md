---
name: "Batch pattern throttling lives on the worker, not the workflow"
description: "All four snippets in BatchCodeViewer.vue show worker-level throttling only; the workflow-level semaphore variant is deliberately not shown."
type: project
---

# Batch pattern throttling lives on the worker, not the workflow

All four snippets (Go, Java, TS, Python) in
`frontend/app/components/BatchCodeViewer.vue` show worker-level
throttling only. The workflow-level sliding-window variant
(semaphore + slot release) is deliberately NOT shown — unified
pedagogy, one idea per demo.

Consequences:

- `BatchInput` has no `Parallelism` field. Do not add one back.
- There is no `workflow.NewSemaphore` / `workflow.Go` slot-release
  goroutine in `workflow.go`. Do not reintroduce them.
- `frontend/server/api/batch/start.post.ts` does not send
  `parallelism` in `args`.
- `frontend/app/components/BatchSlots.vue` still takes a
  `parallelism` prop for the UI-side slot visualization. That is a
  display constant (`PARALLELISM = 4` in `pages/patterns/batch.vue`),
  not a workflow input — leave it alone.

**Why:** Demo-first pedagogy. Mixing two throttling strategies
across languages split the viewer's attention; the unified form
keeps one idea per demo. The numeric cap in Go
(`maxConcurrentActivities = 4`) is chosen to match the
`PARALLELISM = 4` the UI visualizes.

**How to apply:** If asked to "add a semaphore to Go", "pass
parallelism through", or "show the workflow-level variant",
surface this note first and confirm the user really wants to
reintroduce the split before editing. When asked to reintroduce
a semaphore variant, it should live in a separate demo, not
re-split this one. When tuning the cap, change both
`maxConcurrentActivities` and the frontend `PARALLELISM` constant
together.
