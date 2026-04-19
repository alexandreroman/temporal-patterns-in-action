---
name: "Keep CodeViewer snippets in sync across languages"
description: "Any edit to one CodeViewer language snippet must be mirrored in the other three (Go, Java, TypeScript, Python); stepLines stay consistent."
type: feedback
---

# Keep CodeViewer snippets in sync across languages

When editing a code snippet in any
`<Pattern>CodeViewer.vue` component (e.g.
`BatchCodeViewer`, `SagaCodeViewer`), every
language variant (Go, Java, TypeScript, Python)
must be updated together. The snippets represent
the same canonical pattern across SDKs — letting
one drift out of sync is a demo-quality
regression.

**Why:** The user had to explicitly ask to
unify the snippets after I updated only the Go
version of `BatchCodeViewer` (semaphore →
worker-options) and left Java, TypeScript, and
Python still showing the semaphore form. A
single-language change silently breaks the
cross-language parity that the viewer exists to
demonstrate.

**How to apply:**

- Any edit to one snippet triggers a matching
  edit to the other three in the same commit.
- The `stepLines` highlight ranges
  (`dispatch`, `drain`, `summary`, etc.) must
  stay consistent across languages — if one
  language drops a step, the others drop it too.
- If a pattern truly diverges by SDK (so the
  canonical implementation genuinely differs),
  that belongs in a **separate demo**, not a
  split snippet inside a shared viewer.
- Before considering a snippet change done,
  grep the file for stale references
  (`parallelism`, `semaphore`, etc.) across all
  four snippets.
