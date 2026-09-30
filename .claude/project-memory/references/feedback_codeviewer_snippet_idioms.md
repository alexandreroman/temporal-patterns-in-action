---
name: "CodeViewer snippet idioms"
description: "Per-SDK conventions every CodeViewer snippet follows so the four languages read as real, idiomatic code for the same pattern."
type: feedback
---

# CodeViewer snippet idioms

Snippets in `frontend/app/components/*CodeViewer.vue` are
compile-shaped, idiomatic code that mirrors the Go worker:

- **Activity options everywhere.** Go sets
  `workflow.WithActivityOptions` (or `WithLocalActivityOptions`),
  Java declares its stub with `ActivityOptions`/
  `LocalActivityOptions`, TS and Python pass the timeout; values
  match the worker.
- **Workflow failures are Temporal failures.** TS/Java throw
  `ApplicationFailure`, Python raises `ApplicationError`, Go
  returns `temporal.NewNonRetryableApplicationError` where the
  worker does. A plain `Error`/`RuntimeError` only fails the
  workflow task, which retries forever.
- **Java** shows the implementation: Temporal annotations live on
  the `@WorkflowInterface`, impl methods carry `@Override`, and the
  activity stub is named `activities`.
- **Python** passes several activity arguments through
  `args=[...]` (only one positional argument is allowed) and takes
  initial state in `@workflow.init`.
- **TypeScript** uses camelCase (`txId`, `input`, never `in_`).
- **Tabs** are ordered go, java, typescript, python.
- **Highlights** keep the final step lit on
  `progress.workflow.completed`; on failure they show the last
  meaningful step (e.g. saga compensations).

**Why:** the viewer doubles as a guided tour in the reader's SDK;
code that would crash or fail silently in that SDK misteaches the
pattern.

**How to apply:** check new or edited snippets against this list
and against the skill-temporal-developer language references;
keep [[feedback_codeviewer_snippet_sync]] for cross-language and
range alignment.
