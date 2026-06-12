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

**Why:** When a placeholder uses
`id: "pending-prompt"` and the
`agent.user.prompt` event re-renders it with
`id: env.id`, Vue sees a key flip → the leaving
row lingers ~1s because `.msg-row`'s flash
animation is still active, briefly showing two
copies of the same logical message ("two messages
added, then erased, replaced by one"). A single
sentinel key (`USER_PROMPT_KEY`) on both the
placeholder and the event handler eliminates the
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
