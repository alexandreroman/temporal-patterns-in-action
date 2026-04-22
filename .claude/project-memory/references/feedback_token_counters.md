---
name: "Realistic animated token counters"
description: "Demo patterns that show a token counter must emit non-round token counts from the worker per step and animate the UI counter with a useCountTween helper — mirror the agent pattern."
type: feedback
---

# Realistic animated token counters

When a demo pattern surfaces a "Tokens" counter:

- Emit the token count **per business event
  from the worker** using a scripted table of
  realistic, non-round values (e.g. 612, 847,
  186, 213, ...). Do not derive totals on the
  frontend from a flat multiplier like
  `llmCalls * 800 + searches * 100` — that
  produces visibly round, unrealistic numbers.
- Animate the displayed counter with a
  `useCountTween`-style helper
  (requestAnimationFrame, easeOutCubic, honors
  `prefers-reduced-motion`, snaps on reset).
  The reference implementation lives in
  `frontend/app/components/AgentStatePanel.vue`.

**Why:** The user rejected the multi-agent
pattern's first token implementation — a
static display fed by
`TOKENS_PER_LLM_CALL = 800` and
`TOKENS_PER_SEARCH = 100` — because the output
was round and the counter snapped. The agent
pattern already did it correctly: scripted
non-round tokens (742, 918, 1187, 1463, 1724)
emitted on `agent.llm.responded`, plus an
inline `useCountTween` that ticks the number
up smoothly. Keeping every pattern consistent
on this makes the demo feel like a real
LLM-driven run and keeps visual behavior
uniform across pages.

**How to apply:** When adding or reviewing any
pattern that tracks tokens:

1. Put the scripted table in the worker
   (`workers/<pattern>/activities.go`) and
   include the value in the business event
   payload (`map[string]any{..., "tokens": n}`).
2. In the metrics/state panel component,
   accumulate `data.tokens` from each relevant
   event rather than multiplying counters.
3. Use the same `useCountTween` helper as
   `AgentStatePanel.vue` (copy-paste or import)
   to drive the displayed value.
