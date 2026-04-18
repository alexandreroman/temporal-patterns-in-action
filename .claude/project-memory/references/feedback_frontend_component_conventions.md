---
name: "Frontend component conventions"
description: "Generic shell components live in frontend/app/components/; each pattern ships thin <Pattern><Component>.vue wrappers holding only pattern-specific data."
type: feedback
---

# Frontend component conventions

Split reusable UI into two layers:

- **Generic shells** under
  `frontend/app/components/` — e.g.
  `CodeViewer.vue`, `EventStream.vue`,
  `StatusBar.vue`. Take pattern-agnostic props
  (data + callbacks such as `labelFor`) and own
  all presentation.
- **Pattern wrappers** named
  `<Pattern><Component>.vue` — e.g.
  `SagaCodeViewer.vue`, `SagaEventStream.vue`,
  `SagaStatusBar.vue`. Hold only the
  pattern-specific data maps and event-stream
  reducers, then render the generic component.
  Wrappers must never reimplement presentation.

Do not generalize a component until a **second**
pattern actually needs the same shape —
`SagaPipeline` and `SagaArchitecture` stayed
saga-specific on purpose.

**Why:** Keeps pattern-specific knowledge out of
the reusable shells, so scaffolding a new
pattern becomes "write three thin wrappers" with
no risk of regressing existing patterns. Waiting
for a real second consumer avoids premature
abstraction.

**How to apply:** When adding a new pattern,
create `<Pattern>CodeViewer.vue`,
`<Pattern>EventStream.vue`, and
`<Pattern>StatusBar.vue` that consume the
existing generic components. Only promote a new
component to the generic layer when two patterns
share a truly common UI shape. This rule
complements the "Runbook: new pattern" note —
the Nuxt-page step in that runbook should follow
this wrapper convention.
