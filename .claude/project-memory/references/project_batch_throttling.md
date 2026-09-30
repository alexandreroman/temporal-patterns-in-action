---
name: "Batch pattern throttling is an in-workflow sliding window"
description: "BatchProcessingWorkflow bounds in-flight child workflows with a workflow.Selector window; the worker sets no activity concurrency cap."
type: project
---

# Batch pattern throttling is an in-workflow sliding window

`BatchProcessingWorkflow` bounds the number of in-flight child
workflows itself: a sliding window (`workflow.Selector` in Go)
starts a new `ProcessImageWorkflow` only when a slot frees up, then
drains the remaining children before the summary. All four
snippets in `frontend/app/components/BatchCodeViewer.vue` show the
same in-workflow window, each with its SDK's idiom.

Consequences:

- The Batch worker (`workers/batch/cmd/worker/main.go`) sets no
  `MaxConcurrentActivityExecutionSize`; the window bounds the load
  per batch (at most `windowSize` activities), independent of
  `BATCH_WORKER_REPLICAS`. Concurrent batches add up (4 × batches).
- The window size is a Go constant in `workers/batch/workflow.go`
  and matches the UI display constant `PARALLELISM = 4` in
  `frontend/app/pages/patterns/batch.vue`. Change both together.
- The window stays in the parent (no signals, no continue-as-new):
  at `TOTAL = 48` the parent history is far below Temporal's limits
  (2,000 pending children, 51,200 events). The parent costs about
  7-8 history events per item (365 events for 48 items), so the
  10,240-event warning lands around 1,300 items.

**Why:** A worker-level activity cap does not limit how many items
are in flight: every child starts at once, the server holds N open
executions and an N-deep activity backlog, and the cap multiplies
with worker replicas. Temporal's guidance is to cap concurrent
children in the parent; the demo teaches that best practice.

**How to apply:** Keep throttling in the workflow when editing the
Batch pattern. For batches large enough to approach the history or
pending-children limits, the official Sliding Window design
(children signal the parent, parent continues-as-new every window)
is the next step — present it as a separate variant rather than
growing this one.
