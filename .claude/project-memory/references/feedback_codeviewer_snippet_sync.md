---
name: "Keep CodeViewer snippets in sync across languages"
description: "Any edit to a CodeViewer language snippet must be mirrored in the other three AND must recompute highlight ranges (0-indexed stepLines/hl/compLines)."
type: feedback
---

# Keep CodeViewer snippets in sync across languages

When editing a code snippet in any
`<Pattern>CodeViewer.vue` component (e.g.
`BatchCodeViewer`, `SagaCodeViewer`), every
language variant (Go, Java, TypeScript, Python)
must be updated together **and** the highlight
ranges (`stepLines`, `hl`, `compLines`) must be
recomputed against the new line numbers. The
snippets represent the same canonical pattern
across SDKs — letting one drift out of sync,
whether in content or in range alignment, is a
demo-quality regression.

**Why:** Two failure modes share one root cause —
editing a snippet without propagating the change.

1. Updating only the Go snippet (e.g. semaphore →
   worker-options) leaves Java, TypeScript, and
   Python showing the stale form.
2. Editing a snippet's `lines` without recomputing
   the 0-indexed ranges leaves highlights off by a
   line — e.g. a `[37,39]` range misses a
   `return *resp.Plan` that has shifted to line 40.

Both read as demo-quality regressions.

**How to apply:**

- Any edit to one snippet triggers a matching
  edit to the other three in the same commit.
- Ranges are **0-indexed** and the viewer
  compares `idx >= highlight[0] && idx <= highlight[1]`
  where `idx` starts at 0 (displayed line
  numbers are `idx + 1`). After any change to
  the `lines: [...]` array, walk every range in
  that snippet's `stepLines` / `hl` /
  `compLines` and re-verify it covers the
  intended block, not one line above or below.
- The range keys (`dispatch`, `drain`,
  `summary`, etc.) must stay consistent across
  languages — if one language drops a step, the
  others drop it too.
- Each scenario/step key should map to a
  **distinct** range. If two keys must point to
  the same block (e.g. `validate-item` and
  `price-item` both running inside one callback),
  narrow each to its own activity call so the
  highlight visibly moves between events.
- If a pattern truly diverges by SDK (so the
  canonical implementation genuinely differs),
  that belongs in a **separate demo**, not a
  split snippet inside a shared viewer.
- Before considering a snippet change done,
  grep the file for stale references
  (`parallelism`, `semaphore`, etc.) across all
  four snippets.
