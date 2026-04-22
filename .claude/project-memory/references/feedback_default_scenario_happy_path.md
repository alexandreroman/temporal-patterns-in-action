---
name: "Default scenario to happy path when scaffolding a new pattern"
description: "Every new pattern page's scenario selector must default to the happy/clean/success option, not the failure or partial-failure variant."
type: feedback
---

# Default scenario to happy path when scaffolding a new pattern

When scaffolding a new Temporal pattern's Nuxt
page, the reactive form's initial `scenario`
value must be the happy-path option (`"happy"`,
`"clean"`, or whatever the success label is for
that pattern), never the failure, partial, or
retry variant.

**Why:** The user corrected me after I defaulted
`frontend/app/pages/patterns/multi-agent.vue` to
`"partial"` on the assumption that the partial-
failure scenario was the more interesting demo.
The user wants the default to be the clean
success path so a first-time run tells the
pattern's positive story end-to-end; viewers
flip to the failure/partial scenario themselves
when they want to see the resilience behaviour.

**How to apply:** In every new
`frontend/app/pages/patterns/<name>.vue`, set the
`reactive({ scenario: "<happy-option>" })`
initial value to the success scenario. Confirm
by cross-checking the `<option value="…">` list
in the same template — the default must match
whichever option reads as "all succeed" / "happy
path" / "clean". Existing pages already follow
this (saga, batch, agent, encryption); keep it
consistent for future patterns.
