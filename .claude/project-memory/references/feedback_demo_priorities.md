---
name: "Demo-first priorities"
description: "This repo is a live demo — bias code toward visibility, readability, and short inline forms over production concerns like cancellation plumbing, configurability, or defensive edge cases."
type: feedback
---

# Demo-first priorities

This repo is a live demo. Adapt technical
choices to demo priorities over production
priorities:

- **Prefer visibility and readability.** Code
  will be shown on screen to an audience.
  Inline, obvious forms beat clever or
  defensively-correct forms.
- **Prefer simple, predictable behavior.**
  Avoid hidden retries, jitter, or branching
  that obscures what the pattern is
  demonstrating.
- **Skip production-grade robustness by
  default.** Don't add `ctx.Done()` plumbing to
  a sleep, feature flags, env-var knobs,
  configuration toggles, or defensive edge-case
  handling unless the demo itself is *about*
  that piece of complexity.
- **Only add production complexity when the
  demo showcases it.** The retry/heartbeat
  pattern page obviously needs retries and
  heartbeats because that is its point; the
  saga page does not need ctx-cancel handling
  inside its simulated-work sleeps.

**Why:** The user explicitly reminded me of this
after I added a `simulateWork(ctx, d)` helper
in `workers/saga/activities.go` that selects
on `<-time.After(d)` vs `<-ctx.Done()`.
Strictly correct, but heavier than a demo
sleep needs to be — and it adds a helper and
a `context` dance where a one-line
`time.Sleep(d)` would read fine on screen.

**How to apply:** Before writing code in this
repo, ask *"what does the audience need to
see?"*. Prefer the shorter, inline form.
Don't preemptively extract helpers, don't add
cancellation plumbing, don't add config
surface "just in case". Applies repo-wide,
not only to the saga pattern. When a pattern
page legitimately needs production-grade
behavior to make its point, that is the
exception — call it out explicitly in the
change.
