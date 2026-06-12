---
name: "Status bar must not gate content on a Vue out-in transition"
description: "Correctness-bearing UI (StatusBar) must not sit inside <Transition mode='out-in'>; it wedges on hidden tabs because rAF and timers freeze. Use a CSS keyframes animation keyed by the value instead."
type: feedback
---

# Status bar must not gate content on a Vue out-in transition

UI whose displayed value must stay correct (e.g.
`StatusBar.vue`, the generic shell behind every
`<Pattern>StatusBar.vue`) must NOT wrap its content in a
Vue `<Transition mode="out-in">` keyed by the value.

`mode="out-in"` refuses to mount the new keyed child until
the current child's leave transition signals completion —
via `transitionend` OR an explicit `:duration` timer. When
the tab/surface is hidden or occluded, the browser freezes
both `requestAnimationFrame` and timers, so the leave never
finishes and the bar stays pinned on a STALE message even
though the bound prop already updated. `:duration` does not
rescue this — the fallback timer freezes too.

**Why:** observed as "the saga status bar stays on its
initial / penultimate label after the workflow completed,
while the event stream correctly shows it finished". Both
panels read the same `events` ref, so the data was never
the problem — only the transition-gated DOM was stale.

**How to apply:**

- Bind the icon/text directly so they always reflect props.
- For a per-change entry animation, use a CSS `@keyframes`
  animation on an element keyed by the value
  (`<span :key="message">`), like `EventStream.vue`'s
  `.event-row { animation: event-flash … }`. A CSS animation
  never gates DOM mounting, so the value is always current
  even on a hidden tab; it just skips the visual effect.
  Guard it with `@media (prefers-reduced-motion: reduce)`.
- More generally, never gate a value that must stay correct
  behind transition completion. Reserve `mode="out-in"` for
  purely decorative swaps. Relates to the SSR/visibility
  class of frontend gotchas in
  [[feedback_nuxt_ssr_browser_globals]]; StatusBar is a
  generic shell per [[feedback_frontend_component_conventions]].
