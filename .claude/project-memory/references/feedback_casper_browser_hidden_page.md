---
name: "Browser tests: casper load renders a hidden page"
description: "casper browser load drives a hidden page where CSS transitions and rAF freeze; use casper browser open for transition-driven UI flows."
type: feedback
---

# Browser tests: casper load renders a hidden page

`casper browser load <url>` drives the page in the background:
`document.visibilityState` is `"hidden"`, so CSS transitions and
`requestAnimationFrame` stay frozen at their start state. UI that
reveals itself through a transition looks broken there even though
the data is correct — e.g. the agent approval banner stays at
`grid-template-rows: 0px` / `opacity: 0`, its buttons report an
empty `innerText`, and the conversation panel renders blank.

`casper browser open <url>` shows the panel and runs the page
visible; transitions complete and the flows behave as for a user.

`casper browser load` followed immediately by `casper browser
reload` races: the reload hits the previous URL. Poll `casper
browser url --raw` until it matches before reloading or asserting.

**Why:** a hidden-page artefact is easy to misread as a
regression during non-regression runs after a dependency bump.

**How to apply:** for end-to-end UI runs, prefer `casper browser
open`. Before reporting a UI regression found with `load`, check
`document.visibilityState` and the computed style of the element,
and rerun visible. Match buttons by `textContent`, not `innerText`,
when the element may be visually collapsed. Same freeze mechanism
as [[feedback_statusbar_no_outin_transition]].
