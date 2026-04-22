---
name: "Stable Vue keys for placeholder + event-driven items"
description: "In a TransitionGroup, a pending placeholder and the eventual event-driven item must share the same :key, or the CSS flash animation makes both visible during the swap."
type: feedback
---

# Stable Vue keys for placeholder + event-driven items

When a pattern UI shows an optimistic placeholder
(e.g. the user prompt in `AgentConversation.vue`)
that is later replaced by the real event-driven
entry, the placeholder and the real entry must
share the **same Vue `:key`**. If the keys differ,
`<TransitionGroup>` handles the swap as a
leave+enter — and any `animation` on the row class
(we use `.msg-row { animation: msg-flash 1s … }`)
makes Vue wait for the animation to complete
before removing the leaving element, briefly
showing two copies of the same logical message.

**Why:** In Durable AI Agent, clicking *Run
agent* first rendered a placeholder user prompt
with `id: "pending-prompt"`, and the
`agent.user.prompt` event then re-rendered it
with `id: env.id`. Vue saw a key flip → the
leaving row lingered ~1s because `.msg-row`'s
flash animation was still active, producing the
reported "two messages added, then erased,
replaced by one" flicker. Fixing it by using a
single sentinel (`USER_PROMPT_KEY`) in both the
placeholder and the event handler eliminated the
swap entirely.

**How to apply:** Whenever a component prepends
an optimistic row to a `<TransitionGroup>` (or a
`v-for` with transitions) that will later be
supplied by a business event, assign the same
stable key to both paths. The pattern also applies
to any future placeholder we introduce for
saga/batch/encryption/multi-agent — check each
component's `id:` assignment, not just the
visible content.
