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

**Why:** The default must let a first-time run tell
the pattern's positive story end-to-end; viewers
flip to the failure/partial scenario themselves
when they want to see the resilience behaviour. A
failure/partial default (e.g. `"partial"` on the
multi-agent page) misrepresents the pattern at
first glance.

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

**Exception — Priority and Fairness:** the
`priority-fairness` page deliberately defaults
to `"fairness-off"`. The booth narrative for
that pattern *starts from the broken state*
(Mission Critical starves the other tiers) so the
presenter can demonstrate the problem first,
then flip the scenario to `"fairness-on"` to
show how Temporal's `FairnessKey` /
`FairnessWeight` solve it. This is a deliberate
exception; the general rule still holds for every
other pattern.
