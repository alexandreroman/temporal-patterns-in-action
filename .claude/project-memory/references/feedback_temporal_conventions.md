---
name: "Temporal workflow conventions"
description: "Determinism primitives required in workflow code, the workflowcheck gate, and the cross-process contract formed by task queue and workflow type names."
type: feedback
---

# Temporal workflow conventions

- **Workflow determinism:** workflow code must
  be deterministic. Use `workflow.Sleep`,
  `workflow.Now`, `workflow.Go`,
  `workflow.Channel`, `workflow.Selector` —
  never the native `time`, `go`, `chan`, or
  `select` equivalents. No direct I/O, no
  `time.Now()`, no `uuid.NewString()` inside a
  workflow.
- **Run `workflowcheck ./...`** before merging
  workflow changes
  (`make -C workers workflowcheck`).
- **Task queue names and workflow type names
  are a contract** between the Go workers and
  the TypeScript frontend. Renaming either one
  requires a coordinated update on both sides,
  and usually a versioning strategy for
  in-flight executions.

**Why:** Non-determinism corrupts history
replay; silent renames break dispatch across
the language boundary without a compile error.

**How to apply:** When writing workflow code,
substitute the `workflow.*` primitive for the
standard-library call. When proposing a rename
of a `TaskQueue` constant or a workflow type
name, surface the need for a matching change
in `frontend/server/` and flag the impact on
running workflows. Cross-reference the event
architecture memory for the workflow-side
`PublishEvent` pattern.
